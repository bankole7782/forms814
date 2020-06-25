package forms814

import (
	"net/http"
	"github.com/gorilla/mux"
	"io/ioutil"
  "golang.org/x/net/context"
  "cloud.google.com/go/storage"
  "strings"
  "fmt"
  "strconv"
  "html/template"
)

var FILENAME_SEPARATOR = "____"

func serveFileForQF(w http.ResponseWriter, r *http.Request) {
	_, err := GetCurrentUser(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

	filePath := r.FormValue("p")
	parts := strings.Split(filePath, "/")
	tableName := parts[0]

	row, err := FRCL.SearchForOne(fmt.Sprintf(`
		table: f8_document_structures
		where:
			tbl_name = %s
		`, tableName))
	if err != nil {
		errorPage(w, err.Error())
		return
	}

	ds := (*row)["fullname"].(string)
	truthValue, err := DoesCurrentUserHavePerm(r, ds, "read")
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if ! truthValue {
    errorPage(w, "You don't have the read permission for this document structure.")
    return
  }

  ctx := context.Background()
  client, err := storage.NewClient(ctx)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  fr, err := client.Bucket(BucketName).Object(filePath).NewReader(ctx)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  defer fr.Close()

  data, err := ioutil.ReadAll(fr)
	if err != nil {
		errorPage(w, err.Error())
		return
	}

	parts = strings.Split(filePath, FILENAME_SEPARATOR)

	w.Header().Set("Content-Disposition", "attachment; filename=" + parts[1])
	contentType := http.DetectContentType(data)
	w.Header().Set("Content-Type", contentType)
	w.Write(data)
}


func serveJS(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  lib := vars["library"]

  if lib == "jquery" {
    http.ServeFile(w, r, "f8_files/jquery-3.3.1.min.js")
  } else if lib == "autosize" {
    http.ServeFile(w, r, "f8_files/autosize.min.js")
  }
}


func deleteFile(w http.ResponseWriter, r *http.Request) {
  useridInt64, err := GetCurrentUser(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  vars := mux.Vars(r)
  ds := vars["document-structure"]
  docid := vars["id"]
  _, err = strconv.ParseInt(docid, 10, 64)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  detv, err := docExists(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if detv == false {
    errorPage(w, fmt.Sprintf("The document structure %s does not exists.", ds))
    return
  }

  tblName, err := tableName(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  count, err := FRCL.CountRows(fmt.Sprintf(`
  	table: %s
  	where:
  		id = %s
  	`, tblName, docid))
  if err != nil {
  	errorPage(w, err.Error())
  	return
  }
  if count == 0 {
    errorPage(w, fmt.Sprintf("The document with id %s do not exists", docid))
    return
  }

  deletePerm, err := DoesCurrentUserHavePerm(r, ds, "delete")
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  docPerm, err := DoesCurrentUserHavePerm(r, ds, "delete-only-created")
  if err != nil {
    errorPage(w, err.Error())
  }

  row, err := FRCL.SearchForOne(fmt.Sprintf(`
  	table: %s
  	where:
  		id = %s
  	`, tblName, docid))
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  createdBy := (*row)["created_by"].(int64)

  ctx := context.Background()
  client, err := storage.NewClient(ctx)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  if deletePerm || (docPerm && createdBy == useridInt64) {
    toDeleteFileName := (*row)[vars["name"]].(string)
    if err != nil {
      errorPage(w, err.Error())
      return
    }
    err = client.Bucket(BucketName).Object(toDeleteFileName).Delete(ctx)
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    err = FRCL.DeleteFields(fmt.Sprintf(`
	  	table: %s
	  	where:
	  		id = %s
	  		`, tblName, docid), []string{vars["name"], })
    if err != nil {
      errorPage(w, err.Error())
      return
    }
  }

  redirectURL := fmt.Sprintf("/update/%s/%s/", ds, docid)
  http.Redirect(w, r, redirectURL, 307)
}


func completeFilesDelete(w http.ResponseWriter, r *http.Request) {
  useridInt64, err := GetCurrentUser(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  count, err := FRCL.CountRows(fmt.Sprintf(`
  	table: f8_files_for_delete
  	where:
  		created_by = %d
  	`, useridInt64))
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  if count == 0 {
    errorPage(w, "You have nothing to delete.")
    return
  }

  rows, err := FRCL.Search(fmt.Sprintf(`
  	table: f8_files_for_delete
  	where:
  		created_by = %d
  	`, useridInt64))
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  fps := make([]string, 0)
  for _, row := range *rows {
  	fps = append(fps, row["filepath"].(string))
  }

  type Context struct {
    FilePaths []string
  }

  ctx := Context{fps}
  tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "f8_files/complete-files-delete.html"))
  tmpl.Execute(w, ctx)
}


func deleteFileFromBrowser(w http.ResponseWriter, r *http.Request) {
  _, err := GetCurrentUser(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }


  ctx := context.Background()
  client, err := storage.NewClient(ctx)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  fp := r.FormValue("p")
  err = client.Bucket(BucketName).Object(fp).Delete(ctx)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  err = FRCL.DeleteRows(fmt.Sprintf(`
  	table: f8_files_for_delete
  	where:
  		filepath = '%s'
  	`, fp))
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  
  fmt.Fprintf(w, "ok")
}
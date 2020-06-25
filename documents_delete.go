package forms814

import (
  "net/http"
  "github.com/gorilla/mux"
  "strconv"
  "fmt"
  "golang.org/x/net/context"
  "cloud.google.com/go/storage"
  "strings"
  "errors"
  "time"
  "github.com/bankole7782/flaarum"
)


func innerDeleteDocument(r *http.Request, docid string, deleteFile bool) error {
  userIdInt64, err := GetCurrentUser(r)
  if err != nil {
    return err
  }

  vars := mux.Vars(r)
  ds := vars["document-structure"]

  docIdInt64, err := strconv.ParseInt(docid, 10, 64)
  if err != nil {
    return err
  }

  detv, err := docExists(ds)
  if err != nil {
    return err
  }
  if detv == false {
    return errors.New(fmt.Sprintf("The document structure %s does not exists.", ds))
  }

  tblName, err := tableName(ds)
  if err != nil {
    return err
  }

  count, err := FRCL.CountRows(fmt.Sprintf(`
    table: %s
    where:
      id = %d
    `, tblName, docIdInt64))
  if err != nil {
    return err
  }

  if count == 0 {
    return errors.New(fmt.Sprintf("The document with id '%s' do not exists", docid))
  }

  deletePerm, err := DoesCurrentUserHavePerm(r, ds, "delete")
  if err != nil {
    return err
  }
  docPerm, err := DoesCurrentUserHavePerm(r, ds, "delete-only-created")
  if err != nil {
    return err
  }

  // var id int
  // err = SQLDB.QueryRow("select id from qf_document_structures where fullname = ?", ds).Scan(&id)
  // if err != nil {
  //   return err
  // }

  ec, ectv := getEC(ds)

  dds, err := GetDocData(ds)
  if err != nil {
    return err
  }

  var colNames []string
  for _, dd := range dds {
    colNames = append(colNames, dd.Name)
  }

  fData := make(map[string]string)

  arow, err := FRCL.SearchForOne(fmt.Sprintf(`
    table: %s
    where:
      id = %d
    `, tblName, docIdInt64))
  if err != nil {
    return err
  }

  for _, colName := range colNames {
    var data string
    switch dInType := (*arow)[colName].(type) {
    case int64, float64:
      data = fmt.Sprintf("%d", dInType)
    case time.Time:
      data = flaarum.RightDateTimeFormat(dInType)
    case string:
      data = dInType
    case bool:
      data = BoolToStr(dInType)
    }

    fData[colName] = data
  }


  createdBy := (*arow)["created_by"].(int64)

  if deletePerm || (docPerm && createdBy == userIdInt64) {
    if ectv && ec.BeforeDeleteFn != nil {
      ec.BeforeDeleteFn(docIdInt64)
    }

    var ctx context.Context
    var client *storage.Client

    hasForm, err := documentStructureHasForm(ds)
    if hasForm {
      ctx = context.Background()
      client, err = storage.NewClient(ctx)
      if err != nil {
        return err
      }
    }

    for _, dd := range dds {
      if dd.Type == "Table" {
        parts := strings.Split(fData[dd.Name], ",")
        for _, part := range parts {
          ottblName, err := tableName(dd.OtherOptions[0])
          if err != nil {
            return err
          }

          err = FRCL.DeleteRows(fmt.Sprintf(`
            table: %s
            where:
              id = %s
            `, ottblName, part))
          if err != nil {
            return err
          }
        }
      }

      if (deleteFile) {
        if dd.Type == "File" || dd.Type == "Image" {
          client.Bucket(BucketName).Object(fData[dd.Name]).Delete(ctx)
        }        
      } else {
        if dd.Type == "File" || dd.Type == "Image" {
          _, err = FRCL.InsertRowAny("qf_files_for_delete", map[string]interface{} {
            "created_by": userIdInt64, "filepath": fData[dd.Name],
          })
          if err != nil {
            return err
          }
        }
      }

    }

    err = FRCL.DeleteRows(fmt.Sprintf(`
      table: %s
      where:
        id = %s
      `, tblName, docid))
    if err != nil {
      return err
    }
  } else {
    return errors.New("You don't have the delete permission for this document.")
  }

  return nil
}


func deleteDocument(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  docid := vars["id"]
  ds := vars["document-structure"]

  err := innerDeleteDocument(r, docid, true)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  redirectURL := fmt.Sprintf("/list/%s/", ds)
  http.Redirect(w, r, redirectURL, 307)
}


func deleteSearchResults(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  ds := vars["document-structure"]

  detv, err := docExists(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if detv == false {
    errorPage(w, fmt.Sprintf("The document structure %s does not exists.", ds))
    return
  }


  whereFragmentParts, err := parseSearchVariables(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  if len(whereFragmentParts) == 0 {
    errorPage(w, "Your query is empty.")
    return
  }

  tblName, err := tableName(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  rows, err := FRCL.Search(fmt.Sprintf(`
    table: %s
    where:
      %s
    `, tblName, strings.Join(whereFragmentParts, "\n and ")))
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  for _, row := range *rows {
    err = innerDeleteDocument(r, row["id"].(string), false)
    if err != nil {
      errorPage(w, err.Error())
      return
    }
  }

  hasForm, err := documentStructureHasForm(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  var redirectURL string
  if hasForm {
    redirectURL = "/complete-files-delete/"
  } else {
    redirectURL = fmt.Sprintf("/list/%s/", ds)    
  }
  
  http.Redirect(w, r, redirectURL, 307)
}

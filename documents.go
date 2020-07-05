package forms814

import (
  "net/http"
  "fmt"
  "path/filepath"
  "strings"
  "html/template"
  "github.com/gorilla/mux"
  "strconv"
  "html"
  "golang.org/x/net/context"
  "cloud.google.com/go/storage"
  "io"
  "github.com/bankole7782/flaarum"
  "time"
)


func createDocument(w http.ResponseWriter, r *http.Request) {
  userIdInt64, err := GetCurrentUser(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

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

  truthValue, err := DoesCurrentUserHavePerm(r, ds, "create")
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if ! truthValue {
    errorPage(w, "You don't have the create permission for this document structure.")
    return
  }

  ec, ectv := getEC(ds)

  // first check if it passes the extra code validation for this document.
  if ectv && ec.CanCreateFn != nil {
    outString := ec.CanCreateFn()
    if outString != "" {
      errorPage(w, fmt.Sprintf("Document creation process cannot begin because: '%s'", outString))
      return
    }
  }

  arow, err := FRCL.SearchForOne(fmt.Sprintf(`
    table: f8_document_structures
    where:
      fullname = '%s'
    `, ds))
  if err != nil {
    errorPage(w, err.Error())
    return
  }    
  var htStr string
  if ht, ok := (*arow)["help_text"]; ok {
    htStr = strings.Replace(ht.(string), "\n", "<br>", -1)
  }

  ue := func(s string) template.HTML {
    return template.HTML(s)
  }

  dds, err := GetDocData(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  tableFields := make(map[string][]DocData)
  for _, dd := range dds {
    if dd.Type != "Table" {
      continue
    }
    ct := dd.OtherOptions[0]
    tableFields[ct], err = GetDocData(ct)
    if err != nil {
      errorPage(w, err.Error())
      return
    }
  }

  if r.Method == http.MethodGet {
    type Context struct {
      DocumentStructure string
      DDs []DocData
      HelpText string
      UndoEscape func(s string) template.HTML
      TableFields map[string][]DocData
    }

    ctx := Context{ds, dds, htStr, ue, tableFields}
    tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "f8_files/create-document.html"))
    tmpl.Execute(w, ctx)

  } else if r.Method == http.MethodPost {

    r.FormValue("email")

    // first check if it passes the extra code validation for this document.
    if ectv && ec.ValidationFn != nil {
      outString := ec.ValidationFn(r.PostForm)
      if outString != "" {
        errorPage(w, "Exra Code Validation Error: " + outString)
        return
      }
    }

    var ctx context.Context
    var client *storage.Client

    hasForm, err := documentStructureHasForm(ds)
    if hasForm {
      ctx = context.Background()
      client, err = storage.NewClient(ctx)
      if err != nil {
        errorPage(w, err.Error())
        return
      }
    }

    tblName, err := tableName(ds)
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    toInsert := make(map[string]string)
    for _, dd := range dds {
      if dd.Type == "Section Break" {
        continue
      }
      switch dd.Type {
      case "Check":
        var data string
        if r.FormValue(dd.Name) == "on" {
          data = "t"
        } else {
          data = "f"
        }
        toInsert[dd.Name] = data
      case "Table":
        childTableName := dd.OtherOptions[0]
        ddsCT := tableFields[childTableName]
        rowCount := r.FormValue("rows-count-for-" + dd.Name)
        rowCountInt, _ := strconv.Atoi(rowCount)
        rowIds := make([]string, 0)
        for j := 1; j < rowCountInt + 1; j++ {
          jStr := strconv.Itoa(j)
          toInsertCT := make(map[string]string)
          for _, ddCT := range ddsCT {
            tempData := r.FormValue(ddCT.Name + "-" + jStr)

            switch ddCT.Type {
            case "Check":
              var data string
              if tempData == "on" {
                data = "t"
              } else {
                data = "f"
              }
              toInsertCT[ddCT.Name] = data
            default:
              if tempData != "" {
                toInsertCT[ddCT.Name] = html.EscapeString(tempData)
              }
            }
          }
          ctblName, err := tableName(childTableName)
          if err != nil {
            errorPage(w, err.Error())
            return
          }

          lastid, err := FRCL.InsertRowStr(ctblName, toInsertCT)
          if err != nil {
            errorPage(w, err.Error())
            return
          }

          rowIds = append(rowIds, lastid)
        }
        toInsert[dd.Name] = strings.Join(rowIds, ",,,")
      case "File", "Image":
        file, handle, err := r.FormFile(dd.Name)
        if err != nil {
          continue
        }
        defer file.Close()

        var newFileName string
        for {
          randomFileName := filepath.Join(tblName,
            fmt.Sprintf("%s%s%s", untestedRandomString(100),
            FILENAME_SEPARATOR, handle.Filename))

          objHandle := client.Bucket(BucketName).Object(randomFileName)
          _, err := objHandle.NewReader(ctx)
          if err == nil {
            continue
          }

          wc := objHandle.NewWriter(ctx)
          if _, err := io.Copy(wc, file); err != nil {
            errorPage(w, err.Error())
            return
          }
          if err := wc.Close(); err != nil {
            errorPage(w, err.Error())
            return
          }
          newFileName = randomFileName
          break
        }
        toInsert[dd.Name] = newFileName
      default:
        if r.FormValue(dd.Name) != "" {
          toInsert[dd.Name] = r.FormValue(dd.Name)
        }
      }
    }

    toInsert["created"] = flaarum.RightDateTimeFormat(time.Now())
    toInsert["modified"] = flaarum.RightDateTimeFormat(time.Now())
    toInsert["created_by"] = fmt.Sprintf("%d", userIdInt64)

    lastid, err := FRCL.InsertRowStr(tblName, toInsert)
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    // new document extra code
    lastidInt64, err := strconv.ParseInt(lastid, 10, 64)
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    if ectv && ec.AfterCreateFn != nil {
      ec.AfterCreateFn(lastidInt64)
    }

    type Context struct {
      DocumentStructure string
      LastId int64
    }

    tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "f8_files/after-create-msg.html"))
    tmpl.Execute(w, Context{ds, lastidInt64})
  }

}


func updateDocument(w http.ResponseWriter, r *http.Request) {
  userIdInt64, err := GetCurrentUser(r)
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

  readPerm, err := DoesCurrentUserHavePerm(r, ds, "read")
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  rocPerm, err := DoesCurrentUserHavePerm(r, ds, "read-only-created")
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  updatePerm, err := DoesCurrentUserHavePerm(r, ds, "update")
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  deletePerm, err := DoesCurrentUserHavePerm(r, ds, "delete")
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  uocPerm, err := DoesCurrentUserHavePerm(r, ds, "update-only-created")
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  arow, err := FRCL.SearchForOne(fmt.Sprintf(`
    table: %s expand
    where:
      id = %s
    `, tblName, docid))
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  createdBy := (*arow)["created_by"].(int64)

  if ! updatePerm {
    if uocPerm && createdBy == userIdInt64 {
      updatePerm = true
    }
  }

  if ! readPerm {
    if rocPerm {
      if createdBy != userIdInt64 {
        errorPage(w, "You are not the owner of this document so can't read it.")
        return
      }
    } else {
      errorPage(w, "You don't have the read permission for this document structure.")
      return
    }
  }

  dsRow, err := FRCL.SearchForOne(fmt.Sprintf(`
    table: f8_document_structures
    where:
      fullname = '%s'
    `, ds))
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  var htStr string
  if ht, ok := (*dsRow)["help_text"]; ok {
    htStr = strings.Replace(ht.(string), "\n", "<br>", -1)
  }

  ue := func(s string) template.HTML {
    return template.HTML(s)
  }

  docDatas, err := GetDocData(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  type docAndStructure struct {
    DocData
    Data string
  }

  docAndStructureSlice := make([]docAndStructure, 0)
  tableData := make(map[string][][]docAndStructure)

  rowMap := make(map[string]string)
  for k, v := range *arow {
    var data string
    switch dInType := v.(type) {
    case int64, float64:
      data = fmt.Sprintf("%v", dInType)
    case time.Time:
      data = flaarum.RightDateTimeFormat(dInType)
    case string:
      data = dInType
    case bool:
      data = BoolToStr(dInType)
    }

    rowMap[k] = data
  }

  for _, docData := range docDatas {
    if docData.Type == "Section Break" {
      docAndStructureSlice = append(docAndStructureSlice, docAndStructure{docData, ""})
    } else {
      data := rowMap[ docData.Name ]

      docAndStructureSlice = append(docAndStructureSlice, docAndStructure{docData, data})

      if docData.Type == "Table" {
        childTable := docData.OtherOptions[0]
        ctdds, err := GetDocData(childTable)
        if err != nil {
          errorPage(w, err.Error())
          return
        }
        dASSuper := make([][]docAndStructure, 0)

        parts := strings.Split(data, ",,,")
        for _, part := range parts {
          ctblName, err := tableName(childTable)
          if err != nil {
            errorPage(w, err.Error())
            return
          }

          docAndStructureSliceCT := make([]docAndStructure, 0)
          crow, err := FRCL.SearchForOne(fmt.Sprintf(`
            table: %s
            where:
              id = %s
            `, ctblName, part))
          if err != nil {
            errorPage(w, err.Error())
            return
          }

          for _, ctdd := range ctdds {
            crowMap := make(map[string]string)

            for k, v := range *crow {
              var data string
              switch dInType := v.(type) {
              case int64, float64:
                data = fmt.Sprintf("%v", dInType)
              case time.Time:
                data = flaarum.RightDateTimeFormat(dInType)
              case string:
                data = dInType
              case bool:
                data = BoolToStr(dInType)
              }

              crowMap[k] = data
            }

            docAndStructureSliceCT = append(docAndStructureSliceCT, docAndStructure{ctdd, crowMap[ctdd.Name]})
          }
          dASSuper = append(dASSuper, docAndStructureSliceCT)
        }
        tableData[docData.Name] = dASSuper
      }

    }
  }

  created := flaarum.RightDateTimeFormat((*arow)["created"].(time.Time))
  modified := flaarum.RightDateTimeFormat((*arow)["modified"].(time.Time))
  firstname := (*arow)["created_by.firstname"].(string)
  surname := (*arow)["created_by.surname"].(string)
  created_by := (*arow)["created_by"].(int64)

  if r.Method == http.MethodGet {

    dsid, err := getDocumentStructureID(ds)
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    type QFButton struct {
      Name string
      URLPrefix string
    }
    qfbs := make([]QFButton, 0)


    rids, err := GetCurrentUserRolesIds(r)
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    if len(rids) != 0 {
      // qStmt := fmt.Sprintf(`select distinct f8_buttons.name, f8_buttons.url_prefix from f8_buttons inner join f8_btns_and_roles
      // on f8_buttons.id = f8_btns_and_roles.buttonid where f8_btns_and_roles.roleid in ( %s ) and dsid = ?
      // `, strings.Join(rids, " , "))


      rows, err := FRCL.Search(fmt.Sprintf(`
        table: f8_btns_and_roles expand distinct
        fields: buttonid.name buttonid.url_prefix
        where:
          roleid in %s
          and buttonid.dsid = %d
        `, strings.Join(rids, " "), dsid))
      if err != nil {
        errorPage(w, err.Error())
        return
      }
      for _, row := range *rows {
        qfbs = append(qfbs, QFButton{row["buttonid.name"].(string), row["buttonid.url_prefix"].(string)})
      }
    }

    type Context struct {
      Created string
      Modified string
      DocumentStructure string
      DocAndStructures []docAndStructure
      Id string
      FirstName string
      Surname string
      CreatedBy int64
      UpdatePerm bool
      DeletePerm bool
      HelpText string
      UndoEscape func(s string) template.HTML
      TableData map[string][][]docAndStructure
      Add func(x,y int) int
      QFBS []QFButton
    }

    add := func(x, y int) int {
      return x + y
    }

    ctx := Context{created, modified, ds, docAndStructureSlice, docid, firstname, surname,
      created_by, updatePerm, deletePerm, htStr, ue, tableData, add, qfbs}
    tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "f8_files/update-document.html"))
    tmpl.Execute(w, ctx)

  } else if r.Method == http.MethodPost {
    if ! updatePerm {
      errorPage(w, "You don't have permissions to update this document.")
      return
    }

    r.FormValue("email")

    // first check if it passes the extra code validation for this document.
    ec, ectv := getEC(ds)
    if ectv && ec.ValidationFn != nil {
      outString := ec.ValidationFn(r.PostForm)
      if outString != "" {
        errorPage(w, "Exra Code Validation Error: " + outString)
        return
      }
    }

    var ctx context.Context
    var client *storage.Client
    hasForm, err := documentStructureHasForm(ds)
    if hasForm {
      ctx = context.Background()
      client, err = storage.NewClient(ctx)
      if err != nil {
        errorPage(w, err.Error())
        return
      }
    }

    toUpdate := make(map[string]string)
    for _, docAndStructure := range docAndStructureSlice {
      if docAndStructure.DocData.Type == "Table" {
        // delete old table data
        parts := strings.Split(docAndStructure.Data, ",")
        for _, part := range parts {
          ottblName, err := tableName(docAndStructure.DocData.OtherOptions[0])
          if err != nil {
            errorPage(w, "Error getting table name of the table in other options.")
            return
          }

          err = FRCL.DeleteRows(fmt.Sprintf(`
            table: %s
            where:
              id = %s
            `, ottblName, part))
          if err != nil {
            errorPage(w, err.Error())
            return
          }
        }

        // add new table data
        childTableName := docAndStructure.DocData.OtherOptions[0]
        ddsCT, err := GetDocData(childTableName)
        if err != nil {
          errorPage(w, err.Error())
          return
        }

        rowCount := r.FormValue("rows-count-for-" + docAndStructure.DocData.Name)
        rowCountInt, _ := strconv.Atoi(rowCount)
        rowIds := make([]string, 0)
        for j := 1; j < rowCountInt + 1; j++ {

          toInsertCT := make(map[string]string)

          jStr := strconv.Itoa(j)
          for _, ddCT := range ddsCT {
            tempData := r.FormValue(ddCT.Name + "-" + jStr)
            switch ddCT.Type {
            case "Check":
              var data string
              if tempData == "on" {
                data = "t"
              } else {
                data = "f"
              }
              toInsertCT[ddCT.Name] = data
            default:
              if tempData != "" {
                toInsertCT[ddCT.Name] = html.EscapeString(tempData)
              }
            }
          }
          ctblName, err := tableName(childTableName)
          if err != nil {
            errorPage(w, err.Error())
            return
          }

          lastid, err := FRCL.InsertRowStr(ctblName, toInsertCT)
          if err != nil {
            errorPage(w, err.Error())
            return
          }

          rowIds = append(rowIds, lastid)
        }
        toUpdate[docAndStructure.DocData.Name] = strings.Join(rowIds, ",,,")
      } else if docAndStructure.Type == "Image" || docAndStructure.Type == "File" {
        file, handle, err := r.FormFile(docAndStructure.DocData.Name)
        if err != nil {
          continue
        }
        defer file.Close()

        var newFileName string
        for {
          randomFileName := filepath.Join(tblName,
            fmt.Sprintf("%s%s%s", untestedRandomString(100),
            FILENAME_SEPARATOR, handle.Filename))

          objHandle := client.Bucket(BucketName).Object(randomFileName)
          _, err := objHandle.NewReader(ctx)
          if err == nil {
            continue
          }

          wc := objHandle.NewWriter(ctx)
          if _, err := io.Copy(wc, file); err != nil {
            errorPage(w, err.Error())
            return
          }
          if err := wc.Close(); err != nil {
            errorPage(w, err.Error())
            return
          }
          newFileName = randomFileName

          // delete any file that was previously stored.
          client.Bucket(BucketName).Object((*arow)[docAndStructure.DocData.Name].(string)).Delete(ctx)
          break
        }
        toUpdate[docAndStructure.DocData.Name] = newFileName
      } else if docAndStructure.Data != html.EscapeString(r.FormValue(docAndStructure.DocData.Name)) {

        switch docAndStructure.DocData.Type {
        case "Check":
          var data string
          if r.FormValue(docAndStructure.DocData.Name) == "on" {
            data = "t"
          } else {
            data = "f"
          }
          toUpdate[docAndStructure.DocData.Name] = data
        default:
          toUpdate[docAndStructure.DocData.Name] = html.EscapeString(r.FormValue(docAndStructure.DocData.Name))
        }
      }
    }

    toUpdate["modified"] = flaarum.RightDateTimeFormat(time.Now())

    err = FRCL.UpdateRowsStr(fmt.Sprintf(`
      table: %s
      where:
        id = %s
      `, tblName, docid), toUpdate)
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    // post save extra code
    if ectv && ec.AfterUpdateFn != nil {
      docIdInt64, _ := strconv.ParseInt(docid, 10, 64)
      ec.AfterUpdateFn(docIdInt64)
    }

    redirectURL := fmt.Sprintf("/list/%s/", ds)
    http.Redirect(w, r, redirectURL, 307)
  }

}

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
    table: qf_document_structures
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
                data = "\"t\""
              } else {
                data = "\"f\""
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


// func updateDocument(w http.ResponseWriter, r *http.Request) {
//   userIdInt64, err := GetCurrentUser(r)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }

//   vars := mux.Vars(r)
//   ds := vars["document-structure"]
//   docid := vars["id"]
//   _, err = strconv.ParseUint(docid, 10, 64)
//   if err != nil {
//     errorPage(w, err.Error())
//   }

//   detv, err := docExists(ds)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }
//   if detv == false {
//     errorPage(w, fmt.Sprintf("The document structure %s does not exists.", ds))
//     return
//   }

//   readPerm, err := DoesCurrentUserHavePerm(r, ds, "read")
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }
//   rocPerm, err := DoesCurrentUserHavePerm(r, ds, "read-only-created")
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }

//   updatePerm, err := DoesCurrentUserHavePerm(r, ds, "update")
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }
//   deletePerm, err := DoesCurrentUserHavePerm(r, ds, "delete")
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }
//   uocPerm, err := DoesCurrentUserHavePerm(r, ds, "update-only-created")
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }


//   tblName, err := tableName(ds)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }

//   var createdBy uint64
//   sqlStmt := fmt.Sprintf("select created_by from `%s` where id = %s", tblName, docid)
//   err = SQLDB.QueryRow(sqlStmt).Scan(&createdBy)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }

//   if ! updatePerm {
//     if uocPerm && createdBy == userIdInt64 {
//       updatePerm = true
//     }
//   }

//   if ! readPerm {
//     if rocPerm {
//       if createdBy != userIdInt64 {
//         errorPage(w, "You are not the owner of this document so can't read it.")
//         return
//       }
//     } else {
//       errorPage(w, "You don't have the read permission for this document structure.")
//       return
//     }
//   }

//   var count uint64
//   sqlStmt = fmt.Sprintf("select count(*) from `%s` where id = %s", tblName, docid)
//   err = SQLDB.QueryRow(sqlStmt).Scan(&count)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }
//   if count == 0 {
//     errorPage(w, fmt.Sprintf("The document with id %s do not exists", docid))
//     return
//   }

//   var helpText sql.NullString
//   err = SQLDB.QueryRow("select help_text from qf_document_structures where fullname = ?", ds).Scan(&helpText)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }

//   var htStr string
//   if helpText.Valid {
//     htStr = strings.Replace(helpText.String, "\n", "<br>", -1)
//   } else {
//     htStr = ""
//   }

//   ue := func(s string) template.HTML {
//     return template.HTML(s)
//   }

//   docDatas, err := GetDocData(ds)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }

//   type docAndStructure struct {
//     DocData
//     Data string
//   }

//   docAndStructureSlice := make([]docAndStructure, 0)
//   tableData := make(map[string][][]docAndStructure)

//   rows, err := SQLDB.Query(fmt.Sprintf("select * from %s where id = ?", tblName), docid)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }

//   columns, err := rows.Columns()
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }

//   // Make a slice for the values
//   values := make([]sql.RawBytes, len(columns))

//   // rows.Scan wants '[]interface{}' as an argument, so we must copy the
//   // references into such a slice
//   // See http://code.google.com/p/go-wiki/wiki/InterfaceSlice for details
//   scanArgs := make([]interface{}, len(values))
//   for i := range values {
//     scanArgs[i] = &values[i]
//   }

//   for rows.Next() {
//     err = rows.Scan(scanArgs...)
//     if err != nil {
//       errorPage(w, err.Error())
//       return
//     }
//     break
//   }
//   if err = rows.Err(); err != nil {
//     errorPage(w, err.Error())
//     return
//   }

//   rowMap := make(map[string]string)
//   var value string
//   for i, col := range values {
//     if col == nil {
//       value = ""
//     } else {
//       value = html.UnescapeString(string(col))
//     }
//     rowMap[ columns[i] ] = value
//   }

//   for _, docData := range docDatas {
//     if docData.Type == "Section Break" {
//       docAndStructureSlice = append(docAndStructureSlice, docAndStructure{docData, ""})
//     } else {
//       data := rowMap[ docData.Name ]

//       docAndStructureSlice = append(docAndStructureSlice, docAndStructure{docData, data})

//       if docData.Type == "Table" {
//         childTable := docData.OtherOptions[0]
//         ctdds, err := GetDocData(childTable)
//         if err != nil {
//           errorPage(w, err.Error())
//           return
//         }
//         dASSuper := make([][]docAndStructure, 0)

//         parts := strings.Split(data, ",")
//         for _, part := range parts {
//           docAndStructureSliceCT := make([]docAndStructure, 0)
//           for _, ctdd := range ctdds {
//             ctblName, err := tableName(childTable)
//             if err != nil {
//               errorPage(w, err.Error())
//               return
//             }

//             crows, err := SQLDB.Query(fmt.Sprintf("select * from %s where id = ?", ctblName), part)
//             if err != nil {
//               errorPage(w, err.Error())
//               return
//             }

//             ccolumns, err := crows.Columns()
//             if err != nil {
//               errorPage(w, err.Error())
//               return
//             }

//             // Make a slice for the values
//             cvalues := make([]sql.RawBytes, len(ccolumns))

//             // rows.Scan wants '[]interface{}' as an argument, so we must copy the
//             // references into such a slice
//             // See http://code.google.com/p/go-wiki/wiki/InterfaceSlice for details
//             cscanArgs := make([]interface{}, len(cvalues))
//             for i := range cvalues {
//               cscanArgs[i] = &cvalues[i]
//             }

//             for crows.Next() {
//               err = crows.Scan(cscanArgs...)
//               if err != nil {
//                 errorPage(w, err.Error())
//                 return
//               }
//               break
//             }
//             if err = rows.Err(); err != nil {
//               errorPage(w, err.Error())
//               return
//             }

//             crowMap := make(map[string]string)
//             var cvalue string
//             for i, col := range cvalues {
//               if col == nil {
//                 cvalue = ""
//               } else {
//                 cvalue = html.UnescapeString(string(col))
//               }
//               crowMap[ ccolumns[i] ] = cvalue
//             }

//             docAndStructureSliceCT = append(docAndStructureSliceCT, docAndStructure{ctdd, crowMap[ctdd.Name]})
//           }
//           dASSuper = append(dASSuper, docAndStructureSliceCT)
//         }
//         tableData[docData.Name] = dASSuper
//       }

//     }
//   }

//   var created, modified, firstname, surname string
//   var created_by uint64
//   sqlStmt = fmt.Sprintf("select `%[1]s`.created, `%[1]s`.modified, `%[2]s`.firstname, `%[2]s`.surname, `%[2]s`.id ", tblName, UsersTable)
//   sqlStmt += fmt.Sprintf("from `%[1]s` inner join `%[2]s` on `%[1]s`.created_by = `%[2]s`.id where `%[1]s`.id = ?", tblName, UsersTable)
//   err = SQLDB.QueryRow(sqlStmt, docid).Scan(&created, &modified, &firstname, &surname, &created_by)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }

//   if r.Method == http.MethodGet {

//     dsid, err := getDocumentStructureID(ds)
//     if err != nil {
//       errorPage(w, err.Error())
//       return
//     }

//     type QFButton struct {
//       Name string
//       URLPrefix string
//     }
//     qfbs := make([]QFButton, 0)

//     truthValue, err := isUserAdmin(r)
//     if err != nil {
//       errorPage(w, err.Error())
//       return
//     }

//     var (
//       name string
//       urlPrefix string
//     )
//     if truthValue == true {
//       rows, err := SQLDB.Query("select name, url_prefix from qf_buttons where dsid = ?", dsid)
//       if err != nil {
//         errorPage(w, err.Error())
//         return
//       }
//       defer rows.Close()
//       for rows.Next() {
//         err := rows.Scan(&name, &urlPrefix)
//         if err != nil {
//           errorPage(w, err.Error())
//           return
//         }

//         qfbs = append(qfbs, QFButton{name, urlPrefix})
//       }
//       err = rows.Err()
//       if err != nil {
//         errorPage(w, err.Error())
//         return
//       }
//     } else {
//       rids, err := GetCurrentUserRolesIds(r)
//       if err != nil {
//         errorPage(w, err.Error())
//         return
//       }

//       if len(rids) != 0 {
//         qStmt := fmt.Sprintf(`select distinct qf_buttons.name, qf_buttons.url_prefix from qf_buttons inner join qf_btns_and_roles
//         on qf_buttons.id = qf_btns_and_roles.buttonid where qf_btns_and_roles.roleid in ( %s ) and dsid = ?
//         `, strings.Join(rids, " , "))

//         rows, err := SQLDB.Query(qStmt, dsid)
//         if err != nil {
//           errorPage(w, err.Error())
//           return
//         }
//         defer rows.Close()
//         for rows.Next() {
//           err := rows.Scan(&name, &urlPrefix)
//           if err != nil {
//             errorPage(w, err.Error())
//             return
//           }

//           qfbs = append(qfbs, QFButton{name, urlPrefix})
//         }
//         err = rows.Err()
//         if err != nil {
//           errorPage(w, err.Error())
//           return
//         }
//       }
//     }

//     type Context struct {
//       Created string
//       Modified string
//       DocumentStructure string
//       DocAndStructures []docAndStructure
//       Id string
//       FirstName string
//       Surname string
//       CreatedBy uint64
//       UpdatePerm bool
//       DeletePerm bool
//       HelpText string
//       UndoEscape func(s string) template.HTML
//       TableData map[string][][]docAndStructure
//       Add func(x,y int) int
//       QFBS []QFButton
//     }

//     add := func(x, y int) int {
//       return x + y
//     }

//     ctx := Context{created, modified, ds, docAndStructureSlice, docid, firstname, surname,
//       created_by, updatePerm, deletePerm, htStr, ue, tableData, add, qfbs}
//     tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "f8_files/update-document.html"))
//     tmpl.Execute(w, ctx)

//   } else if r.Method == http.MethodPost {
//     if ! updatePerm {
//       errorPage(w, "You don't have permissions to update this document.")
//       return
//     }

//     r.FormValue("email")

//     // first check if it passes the extra code validation for this document.
//     ec, ectv := getEC(ds)
//     if ectv && ec.ValidationFn != nil {
//       outString := ec.ValidationFn(r.PostForm)
//       if outString != "" {
//         errorPage(w, "Exra Code Validation Error: " + outString)
//         return
//       }
//     }

//     var ctx context.Context
//     var client *storage.Client
//     hasForm, err := documentStructureHasForm(ds)
//     if hasForm {
//       ctx = context.Background()
//       client, err = storage.NewClient(ctx)
//       if err != nil {
//         errorPage(w, err.Error())
//         return
//       }
//     }

//     colNames := make([]string, 0)
//     formData := make([]string, 0)
//     for _, docAndStructure := range docAndStructureSlice {
//       if docAndStructure.DocData.Type == "Table" {
//         // delete old table data
//         parts := strings.Split(docAndStructure.Data, ",")
//         for _, part := range parts {
//           ottblName, err := tableName(docAndStructure.DocData.OtherOptions[0])
//           if err != nil {
//             errorPage(w, "Error getting table name of the table in other options.")
//             return
//           }

//           sqlStmt = fmt.Sprintf("delete from `%s` where id = ?", ottblName)
//           _, err = SQLDB.Exec(sqlStmt, part)
//           if err != nil {
//             errorPage(w, err.Error())
//             return
//           }
//         }

//         // add new table data
//         childTableName := docAndStructure.DocData.OtherOptions[0]
//         ddsCT, err := GetDocData(childTableName)
//         if err != nil {
//           errorPage(w, err.Error())
//           return
//         }

//         rowCount := r.FormValue("rows-count-for-" + docAndStructure.DocData.Name)
//         rowCountInt, _ := strconv.Atoi(rowCount)
//         rowIds := make([]string, 0)
//         for j := 1; j < rowCountInt + 1; j++ {
//           colNamesCT := make([]string, 0)
//           formDataCT := make([]string, 0)
//           jStr := strconv.Itoa(j)
//           for _, ddCT := range ddsCT {
//             colNamesCT = append(colNamesCT, ddCT.Name)
//             tempData := r.FormValue(ddCT.Name + "-" + jStr)
//             switch ddCT.Type {
//             case "Text", "Data", "Email", "Read Only", "URL", "Select", "Date", "Datetime":
//               var data string
//               if tempData == "" {
//                 data = "null"
//               } else {
//                 data = fmt.Sprintf("\"%s\"", html.EscapeString(tempData))
//               }
//               formDataCT = append(formDataCT, data)
//             case "Check":
//               var data string
//               if tempData == "on" {
//                 data = "\"t\""
//               } else {
//                 data = "\"f\""
//               }
//               formDataCT = append(formDataCT, data)
//             default:
//               var data string
//               if tempData == "" {
//                 data = "null"
//               } else {
//                 data = html.EscapeString(tempData)
//               }
//               formDataCT = append(formDataCT, data)
//             }
//           }
//           ctblName, err := tableName(childTableName)
//           if err != nil {
//             errorPage(w, err.Error())
//             return
//           }
//           sqlStmt := fmt.Sprintf("insert into `%s`(%s) values (%s)", ctblName,
//             strings.Join(colNamesCT, ", "), strings.Join(formDataCT, ", "))
//           res, err := SQLDB.Exec(sqlStmt)
//           if err != nil {
//             errorPage(w, err.Error())
//             return
//           }
//           lastid, err := res.LastInsertId()
//           if err != nil {
//             errorPage(w, err.Error())
//             return
//           }

//           rowIds = append(rowIds, strconv.FormatInt(lastid, 10))
//         }
//         colNames = append(colNames, docAndStructure.DocData.Name)
//         formData = append(formData, fmt.Sprintf("\"%s\"", strings.Join(rowIds, ",")))

//       } else if docAndStructure.Type == "Image" || docAndStructure.Type == "File" {
//         file, handle, err := r.FormFile(docAndStructure.DocData.Name)
//         if err != nil {
//           continue
//         }
//         defer file.Close()

//         var newFileName string
//         for {
//           randomFileName := filepath.Join(tblName,
//             fmt.Sprintf("%s%s%s", untestedRandomString(100),
//             FILENAME_SEPARATOR, handle.Filename))

//           objHandle := client.Bucket(QFBucketName).Object(randomFileName)
//           _, err := objHandle.NewReader(ctx)
//           if err == nil {
//             continue
//           }

//           wc := objHandle.NewWriter(ctx)
//           if _, err := io.Copy(wc, file); err != nil {
//             errorPage(w, err.Error())
//             return
//           }
//           if err := wc.Close(); err != nil {
//             errorPage(w, err.Error())
//             return
//           }
//           newFileName = randomFileName

//           // delete any file that was previously stored.
//           var datum sql.NullString
//           sqlStmt = fmt.Sprintf("select %s from `%s` where id = ?", docAndStructure.DocData.Name, tblName)
//           SQLDB.QueryRow(sqlStmt, docid).Scan(&datum)
//           if datum.Valid {
//             client.Bucket(QFBucketName).Object(datum.String).Delete(ctx)
//           }
//           break
//         }
//         colNames = append(colNames, docAndStructure.DocData.Name)
//         formData = append(formData, fmt.Sprintf("\"%s\"", newFileName))
//       } else if docAndStructure.Data != html.EscapeString(r.FormValue(docAndStructure.DocData.Name)) {

//         colNames = append(colNames, docAndStructure.DocData.Name)
//         switch docAndStructure.DocData.Type {
//         case "Text", "Data", "Email", "Read Only", "URL", "Select", "Date", "Datetime":
//           data := fmt.Sprintf("\"%s\"", html.EscapeString(r.FormValue(docAndStructure.DocData.Name)))
//           formData = append(formData, data)
//         case "Check":
//           var data string
//           if r.FormValue(docAndStructure.DocData.Name) == "on" {
//             data = "\"t\""
//           } else {
//             data = "\"f\""
//           }
//           formData = append(formData, data)
//         default:
//           formData = append(formData, html.EscapeString(r.FormValue(docAndStructure.DocData.Name)))
//         }
//       }
//     }

//     updatePartStmt := make([]string, 0)
//     updatePartStmt = append(updatePartStmt, "modified = now()")
//     for i := 0; i < len(colNames); i++ {
//       stmt1 := fmt.Sprintf("%s = %s", colNames[i], formData[i])
//       updatePartStmt = append(updatePartStmt, stmt1)
//     }

//     sqlStmt = fmt.Sprintf("update `%s` set %s where id = ?", tblName, strings.Join(updatePartStmt, ", "))
//     _, err = SQLDB.Exec(sqlStmt, docid)
//     if err != nil {
//       errorPage(w, err.Error())
//       return
//     }

//     // post save extra code
//     if ectv && ec.AfterUpdateFn != nil {
//       docidUint64, _ := strconv.ParseUint(docid, 10, 64)
//       ec.AfterUpdateFn(docidUint64)
//     }

//     redirectURL := fmt.Sprintf("/list/%s/", ds)
//     http.Redirect(w, r, redirectURL, 307)
//   }

// }


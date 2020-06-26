package forms814

import (
  "net/http"
  // "fmt"
  "github.com/gorilla/mux"
  "fmt"
  "strings"
  "html/template"
  // "strconv"
)


func lightEditDocumentStructure(w http.ResponseWriter, r *http.Request) {
  truthValue, err := isUserAdmin(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if ! truthValue {
    errorPage(w, "You are not an admin here. You don't have permissions to view this page.")
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

  dsList, err := GetDocumentStructureList("all")
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  row, err := FRCL.SearchForOne(fmt.Sprintf(`
    table: f8_document_structures
    where:
      fullname = '%s'
    `, ds))
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  var helpTextStr string
  childTableBool := (*row)["child_table"].(bool)
  if htAny, ok := (*row)["help_text"]; ok {
    helpTextStr = htAny.(string)
  }

  type Context struct {
    DocumentStructure string
    DocumentStructures string
    OldLabels []string
    NumberofFields int
    OldLabelsStr string
    Add func(x, y int) int
    IsChildTable bool
    HelpText string
  }

  add := func(x, y int) int {
    return x + y
  }

  dsid, err := getDocumentStructureID(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  frows, err := FRCL.Search(fmt.Sprintf(`
    table: f8_fields
    fields: label
    order_by: view_order asc
    where:
      dsid = %d
    `, dsid))
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  labelsList := make([]string, 0)
  for _, row := range *frows {
    labelsList = append(labelsList, row["label"].(string))
  }
  labels := strings.Join(labelsList, ",,,")


  ctx := Context{ds, strings.Join(dsList, ",,,"), labelsList, len(labelsList), labels, add,
    childTableBool, helpTextStr}
  tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "f8_files/light-edit-document-structure.html"))
  tmpl.Execute(w, ctx)
}


func updateDocumentStructureName(w http.ResponseWriter, r *http.Request) {
  truthValue, err := isUserAdmin(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if ! truthValue {
    errorPage(w, "You are not an admin here. You don't have permissions to view this page.")
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

  err = FRCL.UpdateRowsStr(fmt.Sprintf(`
    table: f8_document_structures
    where:
      fullname = '%s'
    `, ds), 
    map[string]string { "fullname": r.FormValue("new-name")},
  )
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  redirectURL := fmt.Sprintf("/view-document-structure/%s/", r.FormValue("new-name"))
  http.Redirect(w, r, redirectURL, 307)
}


// func updateHelpText(w http.ResponseWriter, r *http.Request) {
//   truthValue, err := isUserAdmin(r)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }
//   if ! truthValue {
//     errorPage(w, "You are not an admin here. You don't have permissions to view this page.")
//     return
//   }

//   vars := mux.Vars(r)
//   ds := vars["document-structure"]

//   detv, err := docExists(ds)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }
//   if detv == false {
//     errorPage(w, fmt.Sprintf("The document structure %s does not exists.", ds))
//     return
//   }

//   sqlStmt := "update `qf_document_structures` set help_text = ? where fullname = ?"
//   _, err = SQLDB.Exec(sqlStmt, r.FormValue("updated-help-text"), ds)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }

//   redirectURL := fmt.Sprintf("/view-document-structure/%s/", ds)
//   http.Redirect(w, r, redirectURL, 307)
// }


// func updateFieldLabels(w http.ResponseWriter, r *http.Request) {
//   truthValue, err := isUserAdmin(r)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }
//   if ! truthValue {
//     errorPage(w, "You are not an admin here. You don't have permissions to view this page.")
//     return
//   }

//   vars := mux.Vars(r)
//   ds := vars["document-structure"]

//   detv, err := docExists(ds)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }
//   if detv == false {
//     errorPage(w, fmt.Sprintf("The document structure %s does not exist.", ds))
//     return
//   }

//   r.ParseForm()
//   updateData := make(map[string]string)
//   for i := 1; i < 100; i++ {
//     p := strconv.Itoa(i)
//     if r.FormValue("old-field-label-" + p) == "" {
//       break
//     } else {
//       updateData[ r.FormValue("old-field-label-" + p) ] = r.FormValue("new-field-label-" + p)
//     }
//   }

//   dsid, err := getDocumentStructureID(ds)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }

//   for old, new := range updateData {
//     sqlStmt := "update `qf_fields` set label = ? where dsid=? and label = ?"
//     _, err = SQLDB.Exec(sqlStmt, new, dsid, old)
//     if err != nil {
//       errorPage(w, err.Error())
//       return
//     }
//   }

//   redirectURL := fmt.Sprintf("/view-document-structure/%s/", ds)
//   http.Redirect(w, r, redirectURL, 307)
// }


// func deleteFields(w http.ResponseWriter, r *http.Request) {
//   truthValue, err := isUserAdmin(r)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }
//   if ! truthValue {
//     errorPage(w, "You are not an admin here. You don't have permissions to view this page.")
//     return
//   }

//   vars := mux.Vars(r)
//   ds := vars["document-structure"]

//   detv, err := docExists(ds)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }
//   if detv == false {
//     errorPage(w, fmt.Sprintf("The document structure %s does not exist.", ds))
//     return
//   }

//   dsid, err := getDocumentStructureID(ds)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }

//   tblName, err := tableName(ds)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }

//   r.ParseForm()
//   for _, field := range r.Form["delete-fields-checkbox"] {
//     var mysqlName string
//     err = SQLDB.QueryRow("select name from qf_fields where label = ? and dsid = ?", field, dsid).Scan(&mysqlName)
//     if err != nil {
//       errorPage(w, err.Error())
//       return
//     }

//     sqlStmt := fmt.Sprintf("alter table `%s` drop column %s", tblName, mysqlName)
//     _, err = SQLDB.Exec(sqlStmt)
//     if err != nil {
//       errorPage(w, err.Error())
//       return
//     }

//     _, err = SQLDB.Exec("delete from qf_fields where label = ? and dsid = ?", field, dsid)
//     if err != nil {
//       errorPage(w, err.Error())
//       return
//     }
//   }

//   redirectURL := fmt.Sprintf("/view-document-structure/%s/", ds)
//   http.Redirect(w, r, redirectURL, 307)
// }


// func changeFieldsOrder(w http.ResponseWriter, r *http.Request) {
//   truthValue, err := isUserAdmin(r)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }
//   if ! truthValue {
//     errorPage(w, "You are not an admin here. You don't have permissions to view this page.")
//     return
//   }

//   vars := mux.Vars(r)
//   ds := vars["document-structure"]

//   r.ParseForm()
//   newFieldsOrder := make([]string, 0)
//   for i := 1; i < 100; i ++ {
//     if r.FormValue("el-" + strconv.Itoa(i)) == "" {
//       break
//     }
//     newFieldsOrder = append(newFieldsOrder, r.FormValue("el-" + strconv.Itoa(i)) )
//   }

//   dsid, err := getDocumentStructureID(ds)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }

//   for j, label := range newFieldsOrder {
//     sqlStmt := "update `qf_fields` set view_order = ? where dsid = ? and label = ?"
//     _, err = SQLDB.Exec(sqlStmt, j+1, dsid, label)
//     if err != nil {
//       errorPage(w, err.Error())
//       return
//     }
//   }

//   redirectURL := fmt.Sprintf("/view-document-structure/%s/", ds)
//   http.Redirect(w, r, redirectURL, 307)
// }


// func addFields(w http.ResponseWriter, r *http.Request) {
//   truthValue, err := isUserAdmin(r)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }
//   if ! truthValue {
//     errorPage(w, "You are not an admin here. You don't have permissions to view this page.")
//     return
//   }

//   vars := mux.Vars(r)
//   ds := vars["document-structure"]

//   dsid, err := getDocumentStructureID(ds)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }

//   tblName, err := tableName(ds)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }

//   type QFField struct {
//     label string
//     name string
//     type_ string
//     options string
//     other_options string
//   }

//   var count int
//   err = SQLDB.QueryRow("select count(*) from qf_fields where dsid = ?", dsid).Scan(&count)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }

//   qffs := make([]QFField, 0)
//   r.ParseForm()
//   for i := count + 1; i < 100; i++ {
//     iStr := strconv.Itoa(i)
//     if r.FormValue("label-" + iStr) == "" {
//       break
//     } else {
//       qff := QFField{
//         label: r.FormValue("label-" + iStr),
//         name: r.FormValue("name-" + iStr),
//         type_: r.FormValue("type-" + iStr),
//         options: strings.Join(r.PostForm["options-" + iStr], ","),
//         other_options: r.FormValue("other-options-" + iStr),
//       }
//       qffs = append(qffs, qff)
//     }
//   }

//   stmt, err := SQLDB.Prepare(`insert into qf_fields(dsid, label, name, type, options, other_options, view_order)
//     values(?, ?, ?, ?, ?, ?, ?)`)
//   if err != nil {
//     errorPage(w, err.Error())
//     return
//   }
//   for i, qff := range(qffs) {
//     viewOrder := i + count + 1
//     _, err := stmt.Exec(dsid, qff.label, qff.name, qff.type_, qff.options, qff.other_options, viewOrder)
//     if err != nil {
//       errorPage(w, err.Error())
//       return
//     }

//     if qff.type_ == "Section Break" {
//       continue
//     }

//     sqlStmt := fmt.Sprintf("alter table `%s` add column %s ", tblName, qff.name)
//     var brokenStmt string
//     if qff.type_ == "Big Number" {
//       brokenStmt += "bigint unsigned"
//     } else if qff.type_ == "Check" {
//       brokenStmt += "char(1) default 'f'"
//     } else if qff.type_ == "Date" {
//       brokenStmt += "date"
//     } else if qff.type_ == "Date and Time" {
//       brokenStmt += "datetime"
//     } else if qff.type_ == "Float" {
//       brokenStmt += "float"
//     } else if qff.type_ == "Int" {
//       brokenStmt += "int"
//     } else if qff.type_ == "Link" {
//       brokenStmt += "bigint unsigned"
//     } else if qff.type_ == "Data" || qff.type_ == "Email" || qff.type_ == "URL" || qff.type_ == "Select" || qff.type_ == "Read Only" {
//       brokenStmt += "varchar(255)"
//     } else if qff.type_ == "Text" || qff.type_ == "Table" {
//       brokenStmt += "text"
//     } else if qff.type_ == "File" || qff.type_ == "Image" {
//       brokenStmt += "varchar(255)"
//     }

//     if optionSearch(qff.options, "required") {
//       brokenStmt += " not null"
//     }

//     _, err = SQLDB.Exec(sqlStmt + brokenStmt)
//     if err != nil {
//       errorPage(w, err.Error())
//       return
//     }

//     if optionSearch(qff.options, "unique") {
//       sqlStmt = fmt.Sprintf("alter table `%s` add unique index (%s)", tblName, qff.name)
//       _, err = SQLDB.Exec(sqlStmt)
//       if err != nil {
//         errorPage(w, err.Error())
//         return
//       }
//     }

//     if ! optionSearch(qff.options, "unique") && qff.type_ != "Text" && qff.type_ != "Table" {
//       indexSql := fmt.Sprintf("create index idx_%s on `%s`(%s)", qff.name, tblName, qff.name)
//       _, err := SQLDB.Exec(indexSql)
//       if err != nil {
//         errorPage(w, err.Error())
//         return
//       }

//     }

//     if qff.type_ == "Link" {
//       ottblName, err := tableName(qff.other_options)
//       if err != nil {
//         errorPage(w, err.Error())
//         return
//       }
//       sqlStmt = fmt.Sprintf("alter table `%s` add foreign key (%s) references `%s`(id)", tblName, qff.name, ottblName)
//       _, err = SQLDB.Exec(sqlStmt)
//       if err != nil {
//         errorPage(w, err.Error())
//         return
//       }
//     }
//   }

//   redirectURL := fmt.Sprintf("/view-document-structure/%s/", ds)
//   http.Redirect(w, r, redirectURL, 307)
// }

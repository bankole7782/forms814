package forms814

import (
  "net/http"
  // "fmt"
  "github.com/gorilla/mux"
  "fmt"
  "strings"
  "html/template"
  "strconv"
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


func updateHelpText(w http.ResponseWriter, r *http.Request) {
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
    map[string]string { "help_text": r.FormValue("updated-help-text")},
  )
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  redirectURL := fmt.Sprintf("/view-document-structure/%s/", ds)
  http.Redirect(w, r, redirectURL, 307)
}


func updateFieldLabels(w http.ResponseWriter, r *http.Request) {
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
    errorPage(w, fmt.Sprintf("The document structure %s does not exist.", ds))
    return
  }

  r.ParseForm()
  updateData := make(map[string]string)
  for i := 1; i < 100; i++ {
    p := strconv.Itoa(i)
    if r.FormValue("old-field-label-" + p) == "" {
      break
    } else {
      updateData[ r.FormValue("old-field-label-" + p) ] = r.FormValue("new-field-label-" + p)
    }
  }

  dsid, err := getDocumentStructureID(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  for old, new := range updateData {
    err = FRCL.UpdateRowsStr(fmt.Sprintf(`
      table: f8_fields
      where:
        dsid = %d
        and label = '%s'
      `, dsid, old), 
      map[string]string { "label": new},
    )
    if err != nil {
      errorPage(w, err.Error())
      return
    }
  }

  redirectURL := fmt.Sprintf("/view-document-structure/%s/", ds)
  http.Redirect(w, r, redirectURL, 307)
}


func changeFieldsOrder(w http.ResponseWriter, r *http.Request) {
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

  r.ParseForm()
  newFieldsOrder := make([]string, 0)
  for i := 1; i < 100; i ++ {
    if r.FormValue("el-" + strconv.Itoa(i)) == "" {
      break
    }
    newFieldsOrder = append(newFieldsOrder, r.FormValue("el-" + strconv.Itoa(i)) )
  }

  dsid, err := getDocumentStructureID(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  for j, label := range newFieldsOrder {
    err = FRCL.UpdateRowsAny(fmt.Sprintf(`
      table: f8_fields
      where:
        dsid = %d
        and label = '%s'
      `, dsid, label), 
      map[string]interface{} { "view_order": j + 1},
    )

    if err != nil {
      errorPage(w, err.Error())
      return
    }
  }

  redirectURL := fmt.Sprintf("/view-document-structure/%s/", ds)
  http.Redirect(w, r, redirectURL, 307)
}


func fullEditDocumentStructure(w http.ResponseWriter, r *http.Request) {
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

  docDatas, err := GetDocData(ds)
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

  ctdsList, err := GetDocumentStructureList("only-child-tables")
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  dsList, err := GetDocumentStructureList("all")
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  if r.Method == http.MethodGet {
    type Context struct {
      DocDatas []DocData
      DocumentStructure string
      Add func(x, y int) int
      IsChildTable bool
      HelpText string
      FormatOtherOptions func([]string) string
      ChildTableDocumentStructures string
      DocumentStructures string
    }

    add := func(x, y int) int {
      return x + y
    }

    ffunc := func(x []string) string {
      return strings.Join(x, "\n")
    }

    ctx := Context{docDatas, ds, add, childTableBool, helpTextStr, ffunc, strings.Join(ctdsList, ",,,"), 
      strings.Join(dsList, ",,,")}
    tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "f8_files/full-edit-document-structures.html"))
    tmpl.Execute(w, ctx)    

  } else {

    type QFField struct {
      label string
      name string
      type_ string
      options string
      other_options string
    }

    qffs := make([]QFField, 0)
    r.ParseForm()
    for i := 1; i < 100; i++ {
      iStr := strconv.Itoa(i)
      if r.FormValue("label-" + iStr) == "" {
        break
      } else {
        qff := QFField{
          label: r.FormValue("label-" + iStr),
          name: r.FormValue("name-" + iStr),
          type_: r.FormValue("type-" + iStr),
          options: strings.Join(r.PostForm["options-" + iStr], ","),
          other_options: r.FormValue("other-options-" + iStr),
        }
        qffs = append(qffs, qff)
      }
    }

    dsid, err := getDocumentStructureID(ds)
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    err = FRCL.DeleteRows(fmt.Sprintf(`
      table: f8_fields
      where:
        dsid = %d
      `, dsid))
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    for i, o := range(qffs) {
      toInsertQFFields := map[string]interface{} {
        "dsid": dsid, "label": o.label, "name": o.name, "type": o.type_, "options": o.options,
        "other_options": o.other_options, "view_order": i + 1,
      }
      _, err = FRCL.InsertRowAny("f8_fields", toInsertQFFields)
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

    var stmt string
    if ! childTableBool {
      stmt = fmt.Sprintf(`
      table: %s
      fields:
        created datetime required
        modified datetime required
        created_by int required
      `, tblName)
    } else {
      stmt = fmt.Sprintf(`
        table: %s
        fields:
        `, tblName)
    }

    stmtEnding := ""

    for _, qff := range qffs {
      if qff.type_ == "Section Break" {
        continue
      }
      stmt += "\n" + qff.name + " "
      if qff.type_ == "Check" {
        stmt += "bool"
      } else if qff.type_ == "Date" {
        stmt += "date"
      } else if qff.type_ == "Date and Time" {
        stmt += "datetime"
      } else if qff.type_ == "Float" {
        stmt += "float"
      } else if qff.type_ == "Int" {
        stmt += "int"
      } else if qff.type_ == "Link" {
        stmt += "int"
      } else if qff.type_ == "Data" || qff.type_ == "Email" || qff.type_ == "URL" || qff.type_ == "Select" {
        stmt += "string"
      } else if qff.type_ == "Text" || qff.type_ == "Table" {
        stmt += "text"
      } else if qff.type_ == "File" || qff.type_ == "Image" {
        stmt += "string"
      }
      if optionSearch(qff.options, "required") {
        stmt += " required"
      }

      if optionSearch(qff.options, "unique") {
        stmt += " unique"
      }

      if qff.type_ == "Link" {
        ottblName, err := tableName(qff.other_options)
        if err != nil {
          errorPage(w, err.Error())
          return
        }
        stmtEnding += fmt.Sprintf("\n%s %s on_delete_delete", qff.name, ottblName)
      }
    }
    if ! childTableBool {
      stmtEnding += "\ncreated_by users on_delete_delete"
    }

    stmt += "\n::\nforeign_keys:" + stmtEnding + "\n::"

    err = FRCL.UpdateTableStructure(stmt)
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    redirectURL := fmt.Sprintf("/view-document-structure/%s/", ds)
    http.Redirect(w, r, redirectURL, 307)
  }

}

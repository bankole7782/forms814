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

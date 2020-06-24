package forms814

import (
  "net/http"
  "fmt"
  "strconv"
  "strings"
  "html/template"
  "github.com/gorilla/mux"
)


func newDocumentStructure(w http.ResponseWriter, r *http.Request) {
  truthValue, err := isUserAdmin(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if ! truthValue {
    errorPage(w, "You are not an admin here. You don't have permissions to view this page.")
    return
  }

  if r.Method == http.MethodGet {

    type Context struct {
      DocumentStructures string
      ChildTableDocumentStructures string
    }
    dsList, err := GetDocumentStructureList("all")
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    ctdsList, err := GetDocumentStructureList("only-child-tables")
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    ctx := Context{strings.Join(dsList, ",,,"), strings.Join(ctdsList, ",,,") }

    tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "f8_files/new-document-structure.html"))
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

    tblName, err := newTableName()
    if err != nil {
      errorPage(w, err.Error())
      return
    }
    toInsert := map[string]interface{} {
      "fullname": r.FormValue("ds-name"),
      "tbl_name": tblName,
      "public": false,
    }
    if len(strings.TrimSpace(r.FormValue("help-text"))) != 0 {
      toInsert["help_text"] = r.FormValue("help-text")
    }
    if r.FormValue("child-table") == "on" {
      toInsert["child_table"] = true
    } else {
      toInsert["child_table"] = false
    }

    dsid, err := FRCL.InsertRowAny("qf_document_structures", toInsert)
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    for i, o := range(qffs) {
      toInsertQFFields := map[string]interface{} {
        "dsid": dsid, "label": o.label, "name": o.name, "type": o.type_, "options": o.options,
        "other_options": o.other_options, "view_order": i + 1,
      }
      _, err = FRCL.InsertRowAny("qf_fields", toInsertQFFields)
      if err != nil {
        errorPage(w, err.Error())
        return
      }
    }

    // create actual form data tables, we've only stored the form structure to the database
    var stmt string
    if r.FormValue("child-table") != "on" {
      stmt = fmt.Sprintf(`
      table: %s
      fields:
        created datetime required
        modified datetime required
        created_by int required
        fully_approved bool
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
        stmt += "char(1) default 'f'"
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
    if r.FormValue("child-table") != "on" {
      stmtEnding += "\ncreated_by users on_delete_delete"
    }

    stmt += "\n::\nforeign_keys:" + stmtEnding + "\n::"

    err = FRCL.CreateTable(stmt)
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    redirectURL := fmt.Sprintf("/edit-document-structure-permissions/%s/", r.FormValue("ds-name"))
    http.Redirect(w, r, redirectURL, 307)
  }

}


func listDocumentStructures(w http.ResponseWriter, r *http.Request) {
  truthValue, err := isUserAdmin(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if ! truthValue {
    errorPage(w, "You are not an admin here. You don't have permissions to view this page.")
    return
  }

  type DS struct{
    DSName string
    ChildTable bool
  }

  structDSList := make([]DS, 0)

  rows, err := FRCL.Search(`
    table: qf_document_structures
    fields: fullname child_table
    order_by: fullname asc
    `)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  for _, row := range *rows {
    structDSList = append(structDSList, DS{row["fullname"].(string), row["child_table"].(bool)})
  }

  type Context struct {
    DocumentStructures []DS
  }

  ctx := Context{DocumentStructures: structDSList}

  tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "f8_files/list-document-structures.html"))
  tmpl.Execute(w, ctx)
}


func deleteDocumentStructure(w http.ResponseWriter, r *http.Request) {
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

  row, err := FRCL.SearchForOne(fmt.Sprintf(`
    table: qf_document_structures
    where:
      fullname = '%s'
    `, ds))
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  ctStatus := (*row)["child_table"].(bool)
  if ctStatus {
    dsidsUsingThisCT := make([]int64, 0)

    rows, err := FRCL.Search(fmt.Sprintf(`
      table: qf_fields
      where:
        other_options = '%s'
      `, ds))
    if err != nil {
      errorPage(w, err.Error())
      return
    }
    for _, row := range *rows {
      dsidsUsingThisCT = append(dsidsUsingThisCT, row["dsid"].(int64))
    }

    if len(dsidsUsingThisCT) > 0 {
      m := fmt.Sprintf("This Child Table is in use by the following document structures with id: %s",
        dsidsUsingThisCT)
      errorPage(w, m)
      return
    }
  }

  approvers, err := getApprovers(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  for _, step := range approvers {
    atn, err := getApprovalTable(ds, step)
    if err != nil {
      errorPage(w, "An error occurred getting approval table name.")
      return
    }

    err = FRCL.DeleteTable(atn)
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    err = FRCL.DeleteRows(fmt.Sprintf(`
      table: qf_approvals_tables expand
      where:
        dsid.fullname = '%s'
        and roleid.role = '%s'
      `, ds, step))
    if err != nil {
      errorPage(w, "Error occurred removing record of approval table.")
      return
    }

  }

  tblName, err := tableName(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  hasForm, err := documentStructureHasForm(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  if hasForm {
    dds, err := GetDocData(ds)
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    useridInt64, err := GetCurrentUser(r)
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    for _, dd := range dds {
      if dd.Type == "File" || dd.Type == "Image" {
        rows, err := FRCL.Search(fmt.Sprintf(`
          table: %s
          where:
            created_by = %d
          `, tblName, useridInt64))
        if err != nil {
          errorPage(w, err.Error())
          return
        }

        filepaths := make([]string, 0)
        for _, row := range *rows {
          if fph, ok := row[dd.Name]; ok {
            filepaths = append(filepaths, fph.(string))
          }
        }

        for _, fp := range filepaths {
          _, err = FRCL.InsertRowAny("qf_files_for_delete", 
            map[string]interface{} {"created_by": useridInt64, "filepath": fp})
          if err != nil {
            panic(err)
          }
        }        
      }
    }

  }

  err = FRCL.DeleteTable(tblName)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  err = FRCL.DeleteRows(fmt.Sprintf(`
    table: qf_document_structures
    where:
      fullname = '%s'
    `, ds))
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  var redirectURL string
  if hasForm {
    redirectURL = "/complete-files-delete/?n=l"
  } else {
    redirectURL = "/list-document-structures/"
  }
  http.Redirect(w, r, redirectURL, 307)
}


func viewDocumentStructure(w http.ResponseWriter, r *http.Request) {
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

  row, err := FRCL.SearchForOne(fmt.Sprintf(`
    table: qf_document_structures
    fields: tbl_name id public child_table
    where:
      fullname = '%s'
    `, ds))
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  id := (*row)["id"].(int64)
  tblName := (*row)["tbl_name"].(string)
  publicBool := (*row)["public"].(bool)
  childTableBool := (*row)["child_table"].(bool)

  docDatas, err := GetDocData(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  type Context struct {
    DocumentStructure string
    DocDatas []DocData
    Id int64
    Add func(x, y int) int
    RPS []RolePermissions
    ApproversStr string
    HasApprovers bool
    ChildTable bool
    TableName string
    Public bool
  }

  add := func(x, y int) int {
    return x + y
  }

  rps, err := getRolePermissions(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  approvers, err := getApprovers(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  var hasApprovers bool
  if len(approvers) == 0 {
    hasApprovers = false
  } else {
    hasApprovers = true
  }

  ctx := Context{ds, docDatas, id, add, rps, strings.Join(approvers, ", "), hasApprovers,
    childTableBool, tblName, publicBool}
  tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "f8_files/view-document-structure.html"))
  tmpl.Execute(w, ctx)
}


func editDocumentStructurePermissions(w http.ResponseWriter, r *http.Request) {
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

  if r.Method == http.MethodGet {

    type Context struct {
      DocumentStructure string
      RPS []RolePermissions
      LenRPS int
      Roles []string
    }

    roles, err := GetRoles()
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    rps, err := getRolePermissions(ds)
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    ctx := Context{ds, rps, len(rps), roles}
    tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "f8_files/edit-document-structure-permissions.html"))
    tmpl.Execute(w, ctx)

  } else if r.Method == http.MethodPost {
    r.ParseForm()
    nrps := make([]RolePermissions, 0)
    for i := 1; i < 1000; i++ {
      p := strconv.Itoa(i)
      if r.FormValue("role-" + p) == "" {
        break
      } else {
        if len(r.PostForm["perms-" + p]) == 0 {
          continue
        }
        rp := RolePermissions{r.FormValue("role-" + p), strings.Join(r.PostForm["perms-" + p], ",")}
        nrps = append(nrps, rp)
      }
    }

    dsid, err := getDocumentStructureID(ds)
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    for _, rp := range nrps {
      roleid, err := getRoleId(rp.Role)
      if err != nil {
        errorPage(w, err.Error())
        return
      }
      _, err = FRCL.InsertRowAny("qf_permissions", map[string]interface{} {
        "roleid": roleid, "dsid": dsid, "permissions": rp.Permissions,
      })
      if err != nil {
        errorPage(w, err.Error())
        return
      }
    }

    redirectURL := fmt.Sprintf("/view-document-structure/%s/", ds)
    http.Redirect(w, r, redirectURL, 307)
  }

}

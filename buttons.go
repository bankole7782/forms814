package forms814

import (
  "net/http"
  "html/template"
  "github.com/gorilla/mux"
  "fmt"
)


func createButton(w http.ResponseWriter, r *http.Request) {
  truthValue, err := isUserAdmin(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if ! truthValue {
    errorPage(w, "You are not an admin here. You don't have permissions to view this page.")
    return
  }

  dsList, err := GetDocumentStructureList("not-child-tables")
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  roles, err := GetRoles()
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  if r.Method == http.MethodGet {
    type Context struct {
      DocumentStructureList []string
      Roles []string
    }
    ctx := Context{dsList, roles}

    tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "f8_files/create-button.html"))
    tmpl.Execute(w, ctx)


  } else {
    ds := r.FormValue("ds")
    dsid, err := getDocumentStructureID(ds)
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    btnid, err := FRCL.InsertRowAny("qf_buttons", map[string]interface{} {
      "name": r.FormValue("button_name"),
      "dsid": dsid,
      "url_prefix": r.FormValue("url_prefix"),
    })
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    execRoles := r.PostForm["roles"]

    for _, r := range execRoles {
      roleid, err := getRoleId(r)
      if err != nil {
        errorPage(w, err.Error())
        return
      }
      _, err = FRCL.InsertRowStr("qf_btns_and_roles", map[string]string { 
        "roleid": fmt.Sprintf("%d", roleid), "buttonid": btnid })
      if err != nil {
        errorPage(w, err.Error())
        return
      }
    }

    http.Redirect(w, r, "/list-buttons/", 307)
  }

}


func listButtons(w http.ResponseWriter, r *http.Request) {
  truthValue, err := isUserAdmin(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if ! truthValue {
    errorPage(w, "You are not an admin here. You don't have permissions to view this page.")
    return
  }

  type QFButton struct {
    ButtonId string
    Name string
    DocumentStructure string
    URLPrefix string
    Roles []string
  }
  qfbs := make([]QFButton, 0)

  rows, err := FRCL.Search(`
    table: qf_buttons expand
    order_by: name asc
    `)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  for _, row := range *rows {
    dsName := row["dsid.fullname"].(string)
    buttonId := row["id"].(string)
    rows2, err := FRCL.Search(fmt.Sprintf(`
      table: qf_btns_and_roles expand
      order_by: roleid.role asc
      where:
        buttonid = %s
      `, buttonId))
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    btnRoles := make([]string, 0)
    for _, row2 := range *rows2 {
      btnRoles = append(btnRoles, row2["roleid.role"].(string))
    }

    qfbs = append(qfbs, QFButton{buttonId, row["name"].(string), dsName, row["url_prefix"].(string), btnRoles})
  }

  type Context struct {
    QFBS []QFButton
  }

  ctx := Context{qfbs}
  tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "f8_files/list-buttons.html"))
  tmpl.Execute(w, ctx)
}


func deleteButton(w http.ResponseWriter, r *http.Request) {
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
  bid := vars["id"]

  err = FRCL.DeleteRows(fmt.Sprintf(`
    table: qf_buttons
    where:
      id = %s
    `, bid))
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  http.Redirect(w, r, "/list-buttons/", 307)
}

package forms814

import (
  "github.com/gorilla/mux"
  "net/http"
  "fmt"
  "html/template"
  "strconv"
  "html"
  "strings"
  "math"
  "errors"
)


func rolesView(w http.ResponseWriter, r *http.Request) {
  truthValue, err := isUserAdmin(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if ! truthValue {
    errorPage(w, "You are not an admin here. You don't have permissions to view this page.")
    return
  }

  roles, err := GetRoles()
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  type Context struct {
    Roles []string
    NumberOfRoles int
    RolesStr string
  }
  ctx := Context{roles, len(roles), strings.Join(roles, ",,,")}
  tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "f8_files/roles-view.html"))
  tmpl.Execute(w, ctx)
}


func newRole(w http.ResponseWriter, r *http.Request) {
  truthValue, err := isUserAdmin(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if ! truthValue {
    errorPage(w, "You are not an admin here. You don't have permissions to view this page.")
    return
  }

  roles, err := GetRoles()
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  if r.Method == http.MethodPost {

    rolesRaw := html.EscapeString(r.FormValue("roles"))
    newRoles := strings.Split(strings.TrimSpace(rolesRaw), "\n")

    for _, r := range newRoles {
      r = strings.TrimSpace(r)
      if len(r) == 0 {
        continue
      }

      found := false
      for _, rl := range roles {
        if r == rl {
          found = true
          break
        }
      }

      if ! found {
        _, err := FRCL.InsertRowAny("qf_roles", map[string]interface{} {"role": r})
        if err != nil {
          errorPage(w, err.Error())
          return
        }
      }
    }

    http.Redirect(w, r, "/roles-view/", 307)
  }
}


func renameRole(w http.ResponseWriter, r *http.Request) {
  truthValue, err := isUserAdmin(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if ! truthValue {
    errorPage(w, "You are not an admin here. You don't have permissions to view this page.")
    return
  }

  if r.Method == http.MethodPost {
    err := FRCL.UpdateRowsAny(fmt.Sprintf(`
      table: qf_roles
      where:
        role = '%s'
      `, html.EscapeString(r.FormValue("role-to-rename") )),
      map[string]interface{} { "role": html.EscapeString(r.FormValue("new-name")) },
    )
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    http.Redirect(w, r, "/roles-view/", 307)
  }
}


func deleteRole(w http.ResponseWriter, r *http.Request) {
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
  role := vars["role"]

  err = FRCL.DeleteRows(fmt.Sprintf(`
    table: qf_roles
    where:
      role = '%s'
    `, role))

  http.Redirect(w, r, "/roles-view/", 307)
}


func usersToRolesList(w http.ResponseWriter, r *http.Request) {
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
  page := vars["page"]
  var pageI int64
  if page != "" {
    pageI, err = strconv.ParseInt(page, 10, 64)
    if err != nil {
      errorPage(w, err.Error())
      return
    }
  } else {
    pageI = 1
  }

  count, err := FRCL.CountRows(`
    table: users
    `)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if count == 0 {
    errorPage(w, "You have not defined any users.")
    return
  }

  var itemsPerPage int64 = 50
  startIndex := (pageI - 1) * itemsPerPage
  totalItems := count
  totalPages := math.Ceil( float64(totalItems) / float64(itemsPerPage) )

  type UserData struct {
    UserId int64
    Firstname string
    Surname string
    Roles []string
  }

  uds := make([]UserData, 0)

  rows, err := FRCL.Search(fmt.Sprintf(`
    table: users
    order_by: firstname asc
    start_index: %d
    limit: %d
    `, startIndex, itemsPerPage))
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  for _, row := range *rows {
    userid := row["id"].(int64)

    roles := make([]string, 0)

    for _, id := range Admins {
      if userid == id {
        roles = append(roles, "Administrator")
        break
      }
    }

    for _, id := range Inspectors {
      if userid == id {
        roles = append(roles, "Inspector")
        break
      }
    }

    userRoles, err := getUserRoles(userid)
    if err != nil {
      errorPage(w, err.Error())
      return
    }
    roles = append(roles, userRoles...)
    ud := UserData{userid, row["firstname"].(string), row["surname"].(string), roles}
    uds = append(uds, ud)
  }

  type Context struct {
    UserDatas []UserData
    Pages []int64
  }
  pages := make([]int64, 0)
  for i := int64(0); i < int64(totalPages); i++ {
    pages = append(pages, i+1)
  }

  ctx := Context{uds, pages}
  tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "f8_files/users-to-roles-list.html"))
  tmpl.Execute(w, ctx)
}



func editUserRoles(w http.ResponseWriter, r *http.Request) {
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
  userid := vars["userid"]
  useridInt64, err := strconv.ParseInt(userid, 10, 64)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  count, err := FRCL.CountRows(fmt.Sprintf(`
    table: users
    where:
      id = %s
    `, userid))
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if count == 0 {
    errorPage(w, "The userid does not exist.")
    return
  }

  userRoles, err := getUserRoles(useridInt64)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  if r.Method == http.MethodGet {
    roles, err := GetRoles()
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    row, err := FRCL.SearchForOne(fmt.Sprintf(`
      table: users
      where:
        id = %d
      `, useridInt64))
    if err != nil {
      errorPage(w, err.Error())
      return
    }
    firstname := (*row)["firstname"].(string)
    surname := (*row)["surname"].(string)

    type Context struct {
      UserId string
      UserRoles []string
      Roles []string
      FullName string
    }

    ctx := Context{userid, userRoles, roles, firstname + " " + surname}
    tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "f8_files/edit-user-roles.html"))
    tmpl.Execute(w, ctx)

  } else if r.Method == http.MethodPost {

    r.ParseForm()
    newRoles := r.PostForm["roles"]
    for _, str := range newRoles {
      roleid, err := getRoleId(str)
      if err != nil {
        errorPage(w, err.Error())
        return
      }

      _, err = FRCL.InsertRowAny("qf_user_roles", map[string]interface{} {"userid": useridInt64, "roleid": roleid})
      if err != nil {
        errorPage(w, err.Error())
        return
      }
    }

    http.Redirect(w, r, "/users-to-roles-list/", 307)
  }

}


func removeRoleFromUserInner(role, userid string) error {
  useridInt64, err := strconv.ParseInt(userid, 10, 64)
  if err != nil {
    return err
  }

  count, err := FRCL.CountRows(fmt.Sprintf(`
    table: users
    where:
      id = %s
    `, userid))
  if err != nil {
    return err
  }
  if count == 0 {
    return errors.New("The userid does not exist.")
  }

  roleid, err := getRoleId(role)
  if err != nil {
    return err
  }

  err = FRCL.DeleteRows(fmt.Sprintf(`
    table: qf_user_roles
    where:
      userid = %d
      and roleid = %d
    `, useridInt64, roleid))
  if err != nil {
    return err
  }

  return nil
}

func removeRoleFromUser(w http.ResponseWriter, r *http.Request) {
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
  role := vars["role"]
  userid := vars["userid"]

  err = removeRoleFromUserInner(role, userid)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  redirectURL := fmt.Sprintf("/edit-user-roles/%s/", userid)
  http.Redirect(w, r, redirectURL, 307)
}


func deleteRolePermissions(w http.ResponseWriter, r *http.Request) {
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
  role := vars["role"]
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

  roleid, err := getRoleId(role)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  dsid, err := getDocumentStructureID(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  err = FRCL.DeleteRows(fmt.Sprintf(`
    table: qf_permissions
    where:
      roleid = %d
      and dsid = %d
    `, roleid, dsid))
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  redirectURL := fmt.Sprintf("/edit-document-structure-permissions/%s/", ds)
  http.Redirect(w, r, redirectURL, 307)
}


func userDetails(w http.ResponseWriter, r * http.Request) {
  _, err := GetCurrentUser(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  vars := mux.Vars(r)
  useridToView := vars["userid"]

  count, err := FRCL.CountRows(fmt.Sprintf(`
    table: users
    where:
      id = %s
    `, useridToView))
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if count == 0 {
    errorPage(w, "The userid does not exist.")
    return
  }

  row, err := FRCL.SearchForOne(fmt.Sprintf(`
    table: users
    where:
      id = %s
    `, useridToView))
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  firstname := (*row)["firstname"].(string)
  surname := (*row)["surname"].(string)

  roles := make([]string, 0)

  useridToViewInt64, err := strconv.ParseInt(useridToView, 10, 64)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  for _, id := range Admins {
    if useridToViewInt64 == id {
      roles = append(roles, "Administrator")
      break
    }
  }

  for _, id := range Inspectors {
    if useridToViewInt64 == id {
      roles = append(roles, "Inspector")
      break
    }
  }

  userRoles, err := getUserRoles(useridToViewInt64)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  roles = append(roles, userRoles...)

  type Context struct {
    UserId string
    UserRoles []string
    FullName string
  }

  ctx := Context{useridToView, roles, firstname + " " + surname}
  tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "f8_files/user-details.html"))
  tmpl.Execute(w, ctx)
}


func viewRoleMembers(w http.ResponseWriter, r *http.Request) {
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
  role := vars["role"]

  type UserSummary struct {
    UserId int64
    Firstname string
    Surname string
    Email string
  }

  uss := make([]UserSummary, 0)

  rows, err := FRCL.Search(fmt.Sprintf(`
    table: qf_user_roles expand
    order_by: userid.firstname asc
    where:
      roleid.role = '%s'
    `, role))
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  for _, row := range *rows {
    uss = append(uss, UserSummary{row["userid"].(int64), row["userid.firstname"].(string), 
      row["userid.surname"].(string), row["userid.email"].(string)})
  }

  type Context struct {
    Role string
    UserSummaries []UserSummary
    UsersCount int
  }

  ctx := Context{role, uss, len(uss)}
  tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "f8_files/view-roles-members.html"))
  tmpl.Execute(w, ctx)
}


func removeRoleFromUser2(w http.ResponseWriter, r *http.Request) {
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
  role := vars["role"]
  userid := vars["userid"]

  err = removeRoleFromUserInner(role, userid)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  redirectURL := fmt.Sprintf("/view-role-members/%s/", role)
  http.Redirect(w, r, redirectURL, 307)
}

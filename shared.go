package forms814

import (
  "strings"
  "net/http"
  "fmt"
  "strconv"
  "os"
  "html/template"
  "math/rand"
  "time"
  "runtime"
)

func getBaseTemplate() string {
  if BaseTemplate != "" {
    return BaseTemplate
  } else {
    return "f8_files/bad-base.html"
  }
}


func errorPage(w http.ResponseWriter, msg string) {
  _, fn, line, _ := runtime.Caller(1)
  type Context struct {
    Message template.HTML
    SourceFn string
    SourceLine int
    PANDOLEE_DEVELOPER bool
  }

  var ctx Context
  if os.Getenv("PANDOLEE_DEVELOPER") == "true" {
    msg = strings.ReplaceAll(msg, "\n", "<br>")
    msg = strings.ReplaceAll(msg, "\t", "&nbsp;&nbsp;&nbsp;")
    ctx = Context{template.HTML(msg), fn, line, true}
  } else {
    ctx = Context{template.HTML(msg), fn, line, false}
  }
  tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "f8_files/error-page.html"))
  tmpl.Execute(w, ctx)
}


func isUserAdmin(r *http.Request) (bool, error) {
  userid, err := GetCurrentUser(r)
  if err != nil {
    return false, err
  }
  for _, id := range Admins {
    if userid == id {
      return true, nil
    }
  }
  return false, nil
}


func isUserInspector(r *http.Request) (bool, error) {
  userid, err := GetCurrentUser(r)
  if err != nil {
    return false, err
  }
  for _, id := range Inspectors {
    if userid == id {
      return true, nil
    }
  }
  return false, nil
}


func GetRoles() ([]string, error) {
  strSlice := make([]string, 0)
  rows, err := FRCL.Search(`
  	table: f8_roles
  	order_by: role asc
  	`)
  if err != nil {
    return strSlice, err
  }
  for _, row := range *rows {
  	strSlice = append(strSlice, row["role"].(string))
  }
  return strSlice, nil
}


func getUserRoles(userid int64) ([]string, error) {
	roles := make([]string, 0)

  rows2, err := FRCL.Search(fmt.Sprintf(`
    table: f8_user_roles expand
    where:
      userid = %d
    `, userid))
  if err != nil {
    return roles, err
  }

  for _, row2 := range *rows2 {
    roles = append(roles, row2["roleid.role"].(string))
  }

  return roles, nil
}


func getRoleId(role string) (int64, error) {
	row, err := FRCL.SearchForOne(fmt.Sprintf(`
		table: f8_roles
		where:
			role = '%s'
		`, role))
	if err != nil {
		return 0, err
	}

  idStr := (*row)["id"].(string)
  idInt64, err := strconv.ParseInt(idStr, 10, 64)
  if err != nil {
    return 0, err
  }
  return idInt64, nil
}


func GetDocumentStructureList(situation string) ([]string, error) {
  tempSlice := make([]string, 0)

  var rows *[]map[string]interface{}
  var err error
  if situation == "only-child-tables" {
  	rows, err = FRCL.Search(`
  		table: f8_document_structures
  		fields: fullname
  		where:
  			child_table = t
  		`)
  } else if situation == "all" {
	  rows, err = FRCL.Search(`
	  	table: f8_document_structures
			fields: fullname
	  	`)
  } else if situation == "not-child-tables" {
  	rows, err = FRCL.Search(`
  		table: f8_document_structures
  		fields: fullname
  		where:
  			child_table = f
  		`)
  }
	if err != nil {
		return tempSlice, err
	}


  for _, row := range *rows {
  	tempSlice = append(tempSlice, row["fullname"].(string))
  }

  return tempSlice, nil
}


func untestedRandomString(length int) string {
  var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
  const charset = "abcdefghijklmnopqrstuvwxyz1234567890"

  b := make([]byte, length)
  for i := range b {
    b[i] = charset[seededRand.Intn(len(charset))]
  }
  return string(b)
}


func newTableName() (string, error) {
  for {
    newName := "f8tbl_" + untestedRandomString(3)
    count, err := FRCL.CountRows(fmt.Sprintf(`
    	table: f8_document_structures
    	where:
    		tbl_name = %s
    	`, newName))
    if err != nil {
      return "", err
    }
    if count == 0 {
      return newName, nil
    }
  }
}


func tableName(documentStructure string) (string, error) {
	row, err := FRCL.SearchForOne(fmt.Sprintf(`
		table: f8_document_structures
		fields: tbl_name
		where:
			fullname = '%s'
		`, documentStructure))
	if err != nil {
		return "", err
	}
	return (*row)["tbl_name"].(string), nil
}


func optionSearch(commaSeperatedOptions, option string) bool {
  if commaSeperatedOptions == "" {
    return false
  } else {
    options := strings.Split(commaSeperatedOptions, ",")
    optionsTrimmed := make([]string, 0)
    for _, opt := range options {
      optionsTrimmed = append(optionsTrimmed, strings.TrimSpace(opt))
    }
    for _, value := range optionsTrimmed {
      if option == value {
        return true
      }
    }
    return false
  }
}


func docExists(documentName string) (bool, error) {
  dsList, err := GetDocumentStructureList("all")
  if err != nil {
    return false, err
  }

  for _, value := range dsList {
    if value == documentName {
      return true, nil
    }
  }
  return false, nil
}


func getDocumentStructureID(documentStructure string) (int64, error) {
	row, err := FRCL.SearchForOne(fmt.Sprintf(`
		table: f8_document_structures
		where:
			fullname = '%s'
		`, documentStructure))
	if err != nil {
		return 0, err
	}

  idStr := (*row)["id"].(string)
  idInt64, err := strconv.ParseInt(idStr, 10, 64)
  if err != nil {
    return 0, err
  }
	return idInt64, nil
}


type DocData struct {
  Label string
  Name string
  Type string
  Required bool
  Unique bool
  ReadOnly bool
  OtherOptions []string
}


func GetDocData(documentStructure string) ([]DocData, error) {
  dds := make([]DocData, 0)
  dsid, err := getDocumentStructureID(documentStructure)
  if err != nil {
    return dds, err
  }

  rows, err := FRCL.Search(fmt.Sprintf(`
  	table: f8_fields
  	order_by: view_order asc
  	where:
  		dsid = %d
  	`, dsid))
  if err != nil {
    return dds, err
  }
  for _, row := range *rows {
    var label, name, type_, options, otherOptions string

    label = row["label"].(string)
    name = row["name"].(string)
    type_ = row["type"].(string)
    if op, ok := row["options"]; ok {
    	options = op.(string)
    }
    if oo, ok := row["other_options"]; ok {
    	otherOptions = oo.(string)
    }
    var required, unique, readonly bool
    if optionSearch(options, "required") {
      required = true
    }
    if optionSearch(options, "unique") {
      unique = true
    }
    if optionSearch(options, "readonly") {
      readonly = true
    }
    otherOptionsOk := make([]string, 0)
    for _, otherOption := range strings.Split(otherOptions, "\n") {
      otherOptionsOk = append(otherOptionsOk, strings.TrimSpace(otherOption))
    }
    dd := DocData{label, name, type_, required, unique, readonly, otherOptionsOk}
    dds = append(dds, dd)
  }

  return dds, nil
}



type RolePermissions struct {
  Role string
  Permissions string
}


func getRolePermissions(documentStructure string) ([]RolePermissions, error) {
  rps := make([]RolePermissions, 0)
  rows, err := FRCL.Search(fmt.Sprintf(`
  	table: f8_permissions expand
  	where:
  		dsid.fullname = '%s'
  	`, documentStructure))
  if err != nil {
    return rps, err
  }
  for _, row := range *rows {
    rps = append(rps, RolePermissions{row["roleid.role"].(string), row["permissions"].(string)})
  }
  return rps, nil
}


func documentStructureHasForm(documentStructure string) (bool, error) {
  count, err := FRCL.CountRows(fmt.Sprintf(`
  	table: f8_fields expand
  	where:
  		dsid.fullname = %s
  		and type in File Image
  	`, documentStructure))
  if err != nil {
  	return false, err
  }

  ret := false
  if count > 0 {
    ret = true
  }
  return ret, nil
}


func DoesCurrentUserHavePerm(r *http.Request, documentStructure, permission string) (bool, error) {
  state, err := publicState(documentStructure)
  if err != nil {
    return false, err
  }
  if state && permission == "read" {
    return true, nil
  }

  adminTruth, err := isUserAdmin(r)
  if err != nil {
    return false, err
  }
  if err == nil && adminTruth {
    return true, nil
  }

  inspectorTruth, err := isUserInspector(r)
  if err == nil && inspectorTruth && permission == "read" {
    return true, nil
  }

  userid, err := GetCurrentUser(r)
  if err != nil {
    return false, err
  }

  rows, err := FRCL.Search(fmt.Sprintf(`
  	table: f8_user_roles
  	where:
  		userid = %d
  	`, userid))
  if err != nil {
    return false, err
  }

  dsid, err := getDocumentStructureID(documentStructure)
  if err != nil {
    return false, err
  }

  for _, row := range *rows {
  	rid := row["roleid"].(int64)
    count, err := FRCL.CountRows(fmt.Sprintf(`
    	table: f8_permissions
    	where:
    		dsid = %d
    		and roleid = %d
    	`, dsid, rid))
    if err != nil {
      return false, err
    }
    if count == 0 {
      continue
    }
    arow, err := FRCL.SearchForOne(fmt.Sprintf(`
    	table: f8_permissions
    	where:
    		dsid = %d
    		and roleid = %d
    	`, dsid, rid))
    if err != nil {
      return false, err
    }

    permissions := (*arow)["permissions"].(string)
    if optionSearch(permissions, permission) {
      return true, nil
    }
  }

  return false, nil
}


func getEC(documentStructure string) (ExtraCode, bool) {
  dsid, err := getDocumentStructureID(documentStructure)
  if err != nil {
    return ExtraCode{}, false
  }


  ec, ok := ExtraCodeMap[dsid]
  if ok {
    return ec, true
  }
  return ExtraCode{}, false
}


func BoolToStr(b bool) string {
  if b {
    return "t"
  } else {
    return "f"
  }
}


func GetCurrentUserRolesIds(r *http.Request) ([]string, error) {
  userid, err := GetCurrentUser(r)
  if err != nil {
    return nil, err
  }

  rows, err := FRCL.Search(fmt.Sprintf(`
    table: f8_user_roles
    where:
      userid = %s
    `, userid))
  if err != nil {
    return nil, err
  }

  rids := make([]string, 0)
  for _, row := range *rows {
    rids = append(rids, fmt.Sprintf("%d", row["roleid"].(int)))
  }
  return rids, nil
}


func getUserTimeZoneSuffix(userid int64) (string, error) {
  fmt.Println(userid)
  row, err := FRCL.SearchForOne(fmt.Sprintf(`
    table: users
    where:
      id = %d
    `, userid))
  if err != nil {
    return "", err
  }

  fmt.Println("found a row")

  loc, err := time.LoadLocation((*row)["timezone"].(string))
  if err != nil {
    return "", err
  }
  tzname, _ := time.Now().In(loc).Zone()
  return tzname, nil
}

package forms814

import (
  "net/http"
  "fmt"
  "github.com/gorilla/mux"
  "html/template"
  "html"
  "math"
  "strconv"
  "time"
  "github.com/bankole7782/flaarum"
  "strings"
)


func innerListDocuments(w http.ResponseWriter, r *http.Request, tblName, whereFragment, countStmt, listType string) {
  userIdInt64, err := GetCurrentUser(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  vars := mux.Vars(r)
  ds := vars["document-structure"]
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

  detv, err := docExists(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if detv == false {
    errorPage(w, fmt.Sprintf("The document structure %s does not exists.", ds))
    return
  }

  count, err := FRCL.CountRows(countStmt)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if count == 0 {
    cperm, err := DoesCurrentUserHavePerm(r, ds, "create")
    if err != nil {
      errorPage(w, err.Error())
      return
    }
    type Context struct {
      DocumentStructure string
      CreatePerm bool
      ListType string
    }
    tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "f8_files/suggest-create-document.html"))
    tmpl.Execute(w, Context{ds, cperm, listType})
    return
  }

  type ColLabel struct {
    Col string
    Label string
  }

  colNames := make([]ColLabel, 0)
  colNames = append(colNames, ColLabel{"needs_update", "Needs Update"})

  dsid, err := getDocumentStructureID(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  frows, err := FRCL.Search(fmt.Sprintf(`
    table: f8_fields
    order_by: view_order asc
    limit: 3
    where:
      dsid = %d
      and type nin 'Section Break' File Image Table
    `, dsid))
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  for _, row := range *frows {
    colName := row["name"].(string)
    label := row["label"].(string)
    colNames = append(colNames, ColLabel{colName, label})
  }

  colNames = append(colNames, ColLabel{"created", "Creation DateTime"}, ColLabel{"created_by", "Created By"})

  var itemsPerPage int64 = 50
  startIndex := (pageI - 1) * itemsPerPage
  totalItems := count
  totalPages := math.Ceil( float64(totalItems) / float64(itemsPerPage) )


  var orderByFragment string
  if r.FormValue("order_by") != "" {
    // get db name of order_by
    var dbName string
    orderBy := html.EscapeString( r.FormValue("order_by") )
    if orderBy == "Created By" {
      dbName = "created_by"
    } else if orderBy == "Creation DateTime" {
      dbName = "created"
    } else if orderBy == "Modification DateTime" {
      dbName = "modified"
    } else {
      row1, err := FRCL.SearchForOne(fmt.Sprintf(`
        table: f8_fields
        where:
          dsid = %d
          and label = '%s'
        `, dsid, orderBy))
      if err != nil {
        errorPage(w, err.Error())
        return
      }

      dbName = (*row1)["name"].(string)
    }

    var direction string
    if r.FormValue("direction") == "Ascending" {
      direction = "asc"
    } else {
      direction = "desc"
    }

    orderByFragment += fmt.Sprintf(" %s %s", dbName, direction)
  } else {
    orderByFragment += " id desc"
  }

  uocPerm, err1 := DoesCurrentUserHavePerm(r, ds, "update-only-created")
  docPerm, err2 := DoesCurrentUserHavePerm(r, ds, "delete-only-created")
  if err1 != nil || err2 != nil {
    errorPage(w, "Error occured while determining if the user have read permission for this page.")
    return
  }


  type ColAndData struct {
    ColName string
    Data string
  }

  type Row struct {
    Id uint64
    ColAndDatas []ColAndData
    RowUpdatePerm bool
    RowDeletePerm bool
  }

  myRows := make([]Row, 0)

  rows, err := FRCL.Search(fmt.Sprintf(`
    table: %s
    order_by: %s
    limit: %d
    start_index: %d
    where:
      %s
    `, tblName, orderByFragment, itemsPerPage, startIndex, whereFragment))
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  currentVersionNum, err := FRCL.GetCurrentTableVersionNum(tblName)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  for _, rowMapItem := range *rows {
    colAndDatas := make([]ColAndData, 0)
    for _, colLabel := range colNames {
      if colLabel.Col == "needs_update" {
        var data string
        if currentVersionNum == rowMapItem["_version"].(int64) {
          data = "no"
        } else {
          data = "yes"
        }
        colAndDatas = append(colAndDatas, ColAndData{colLabel.Label, data})
      } else {
        var data string
        switch dInType := rowMapItem[colLabel.Col].(type) {
        case int64, float64:
          data = fmt.Sprintf("%v", dInType)
        case time.Time:
          data = flaarum.RightDateTimeFormat(dInType)
        case string:
          data = dInType
        case bool:
          data = BoolToStr(dInType)
        }

        colAndDatas = append(colAndDatas, ColAndData{colLabel.Label, data})
      }

    }

    rup := false
    rdp := false
    createdBy := rowMapItem["created_by"].(int64)
    if err != nil {
      errorPage(w, err.Error())
      return
    }
    if createdBy == userIdInt64 && uocPerm {
      rup = true
    }
    if createdBy == userIdInt64 && docPerm {
      rdp = true
    }
    rid, err := strconv.ParseUint(rowMapItem["id"].(string), 10, 64)
    if err != nil {
      errorPage(w, err.Error())
      return
    }
    myRows = append(myRows, Row{rid, colAndDatas, rup, rdp})

  }


  type Context struct {
    DocumentStructure string
    ColNames []string
    MyRows []Row
    CurrentPage int64
    Pages []int64
    CreatePerm bool
    UpdatePerm bool
    DeletePerm bool
    ListType string
    OrderColumns []string
    Count int64
  }

  pages := make([]int64, 0)
  for i := int64(0); i < int64(totalPages); i++ {
    pages = append(pages, i+1)
  }

  tv1, err1 := DoesCurrentUserHavePerm(r, ds, "create")
  tv2, err2 := DoesCurrentUserHavePerm(r, ds, "update")
  tv3, err3 := DoesCurrentUserHavePerm(r, ds, "delete")
  if err1 != nil || err2 != nil || err3 != nil {
    errorPage(w, "An error occurred when getting permissions of this document structure for this user.")
    return
  }

  colNamesList := make([]string, 0)
  for _, colLabel := range colNames {
    colNamesList = append(colNamesList, colLabel.Label)
  }

  frows2, err := FRCL.Search(fmt.Sprintf(`
    table: f8_fields
    order_by: view_order asc
    where:
      dsid = %d
      and type nin 'Section Break' 'File' 'Image' 'Table'
    `, dsid))
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  allColumnLabels := make([]string, 0)
  for _, row := range *frows2 {
    allColumnLabels = append(allColumnLabels, row["label"].(string))
  }

  ctx := Context{ds, colNamesList, myRows, pageI, pages, tv1, tv2, tv3, listType, allColumnLabels, count}
  tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "f8_files/list-documents.html"))
  tmpl.Execute(w, ctx)
}


func listDocuments(w http.ResponseWriter, r *http.Request) {
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

  tv1, err := DoesCurrentUserHavePerm(r, ds, "read")
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  tv2, err := DoesCurrentUserHavePerm(r, ds, "read-only-created")
  if err != nil {
    errorPage(w, err.Error())
  }

  if ! tv1 && ! tv2 {
    errorPage(w, "You don't have read permission for this document structure.")
    return
  }


  tblName, err := tableName(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  var whereFragment string
  var countStmt string

  if tv1 {
    countStmt = fmt.Sprintf(`
      table: %s
      `, tblName)    
  } else {
    countStmt = fmt.Sprintf(`
      table: %s
      where:
        created_by = %d
      `, tblName, userIdInt64)
    whereFragment = fmt.Sprintf("created_by = %d\n", tblName, userIdInt64)
  }

  innerListDocuments(w, r, tblName, whereFragment, countStmt, "true-list")
  return
}


func searchDocuments(w http.ResponseWriter, r *http.Request) {
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

  tv1, err := DoesCurrentUserHavePerm(r, ds, "read")
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  tv2, err := DoesCurrentUserHavePerm(r, ds, "read-only-created")
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  if ! tv1 && ! tv2 {
    errorPage(w, "You don't have the read permission for this document structure.")
    return
  }

  dds, err := GetDocData(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  type Context struct {
    DocumentStructure string
    DDs []DocData
    FullReadAccess bool
  }
  ctx := Context{ds, dds, tv1}
  tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "f8_files/search-documents.html"))
  tmpl.Execute(w, ctx)
}


func parseSearchVariables(r *http.Request) ([]string, error) {
  vars := mux.Vars(r)
  ds := vars["document-structure"]

  dds, err := GetDocData(ds)
  if err != nil {
    return nil, err
  }

  whereFragmentParts := make([]string, 0)
  for _, dd := range dds {
    if dd.Type == "Section Break" || dd.Type == "Image" || dd.Type == "File" {
      continue
    }
    if r.FormValue(dd.Name) == "" {
      continue
    }

    switch dd.Type {
    case "Text", "Data", "Email", "Read Only", "URL", "Select", "Date", "Datetime":
      data := fmt.Sprintf("'%s'", html.EscapeString(r.FormValue(dd.Name)))
      whereFragmentParts = append(whereFragmentParts, dd.Name + " = " + data)
    case "Check":
      var data string
      if r.FormValue(dd.Name) == "on" {
        data = "t"
      } else {
        data = "f"
      }
      whereFragmentParts = append(whereFragmentParts, dd.Name + " = " + data)
    default:
      data := html.EscapeString(r.FormValue(dd.Name))
      whereFragmentParts = append(whereFragmentParts, dd.Name + " = " + data)
    }
  }

  if r.FormValue("created_by") != "" {
    whereFragmentParts = append(whereFragmentParts, "created_by = " + html.EscapeString(r.FormValue("created_by")))
  }
  if r.FormValue("creation-year") != "" {
    whereFragmentParts = append(whereFragmentParts, "created_year = " + html.EscapeString(r.FormValue("creation-year")))
  }
  if r.FormValue("creation-month") != "" {
    whereFragmentParts = append(whereFragmentParts, "created_month = " + html.EscapeString(r.FormValue("creation-month")))
  }

  return whereFragmentParts, nil
}


func searchResults(w http.ResponseWriter, r *http.Request) {
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

  tv1, err := DoesCurrentUserHavePerm(r, ds, "read")
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  tv2, err := DoesCurrentUserHavePerm(r, ds, "read-only-created")
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  if ! tv1 && ! tv2 {
    errorPage(w, "You don't have the read permission for this document structure.")
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

  if tv2 {
    whereFragmentParts = append(whereFragmentParts, fmt.Sprintf("created_by = %d", userIdInt64))
  }

  countStmt := fmt.Sprintf(`
    table: %s
    where:
      %s
    `, tblName, strings.Join(whereFragmentParts, "\nand "))

  whereFragment := strings.Join(whereFragmentParts, "\nand ")
  innerListDocuments(w, r, tblName, whereFragment, countStmt, "search-list")
  return
}

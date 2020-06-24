package forms814

import (
  "fmt"
  "net/http"
  "github.com/gorilla/mux"
  "net/url"
  "strings"
  // "html/template"
  "github.com/bankole7782/flaarum"
  "github.com/bankole7782/flaarum/flaarum_shared"
)

var FRCL flaarum.Client
var Admins []int64
var Inspectors []int64
var GetCurrentUser func(r *http.Request) (int64, error)
var BaseTemplate string

type ExtraCode struct {
  ValidationFn func(postForm url.Values) string
  AfterCreateFn func(id int64)
  AfterUpdateFn func(id int64)
  BeforeDeleteFn func(id int64)
  CanCreateFn func() string
}

var ExtraCodeMap = make(map[int64]ExtraCode)

var BucketName string


func findIn(container []string, toFind string) int {
  for i, inContainer := range container {
    if inContainer == toFind {
      return i
    }
  }
  return -1
}


// lifted from flaarum
func formatTableStruct(tableStruct flaarum_shared.TableStruct) string {
  stmt := "table: " + tableStruct.TableName + "\n"
  stmt += "table-type: " + tableStruct.TableType + "\n"
  stmt += "fields:\n"
  for _, fieldStruct := range tableStruct.Fields {
    stmt += "\t" + fieldStruct.FieldName + " " + fieldStruct.FieldType
    if fieldStruct.Required {
      stmt += " required"
    }
    if fieldStruct.Unique {
      stmt += " unique"
    }
    stmt += "\n"
  }
  stmt += "::\n"
  if len(tableStruct.ForeignKeys) > 0 {
    stmt += "foreign_keys:\n"
    for _, fks := range tableStruct.ForeignKeys {
      stmt += "\t" + fks.FieldName + " " + fks.PointedTable + " " + fks.OnDelete + "\n"
    }
    stmt += "::\n"
  }

  if len(tableStruct.UniqueGroups) > 0 {
    stmt += "unique_groups:\n"
    for _, ug := range tableStruct.UniqueGroups {
      stmt += "\t" + strings.Join(ug, " ") + "\n"
    }
    stmt += "::\n"
  }

  return stmt
}

func createOrUpdateTable(stmt string) error {
	tables, err := FRCL.ListTables()
	if err != nil {
		return err
	}

	tableStruct, err := flaarum_shared.ParseTableStructureStmt(stmt)
	if err != nil {
		return err
	}
	if findIn(tables, tableStruct.TableName) == -1 {
		// table doesn't exist
		err = FRCL.CreateTable(stmt)
		if err != nil {
			return err
		}
	} else {
		// table exists check if it needs update
    currentVersionNum, err := FRCL.GetCurrentTableVersionNum(tableStruct.TableName)
    if err != nil {
      return err
    }

		oldStmt, err := FRCL.GetTableStructure(tableStruct.TableName, currentVersionNum)
		if err != nil {
			return err
		}

		if oldStmt != formatTableStruct(tableStruct) {
			err = FRCL.UpdateTableStructure(stmt)
			if err != nil {
				return err
			}
		}

	}
	return nil
}


func forms814Setup(w http.ResponseWriter, r *http.Request) {
  if err := FRCL.Ping(); err != nil {
    errorPage(w, err.Error())
    return
  }

  if Admins == nil {
    errorPage(w, "You have not set the \"forms814.Admins\". Please set this to a list of ids (in int64) of the Admins of this site.")
    return
  }

  if GetCurrentUser == nil {
    errorPage(w, "You must set the \"forms814.GetCurrentUser\". Please set this variable to a function with signature func(r *http.Request) (int64, error).")
    return
  }

  if BucketName == "" {
    errorPage(w, "You must set the \"forms814.BucketName\". Create a bucket on google cloud and set it to this variable.")
    return
  }

  // create forms general table
  err := createOrUpdateTable(`
  	table: qf_document_structures
  	fields:
  		fullname string required unique
  		tbl_name string required unique
  		child_table bool
  		approval_steps text
  		help_text text
  		public bool
  	::
  	`)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  // create fields table
  err = createOrUpdateTable(`
  	table: qf_fields
  	fields:
  		dsid int required
  		label string required
  		name string required
  		type string required
  		options string
  		other_options string
  		view_order int
  	::
  	foreign_keys:
  		dsid qf_document_structures on_delete_delete
  	::
  	unique_groups:
  		dsid name
  	::
  	`)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  err = createOrUpdateTable(`
  	table: qf_roles
  	fields:
  		role string required unique
  	::
  	`)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  err = createOrUpdateTable(`
  	table: qf_approvals_tables
  	fields:
  		dsid int required
  		roleid int required
  		tbl_name string required unique
  	::
  	foreign_keys:
  		dsid qf_document_structures on_delete_delete
  		roleid qf_roles on_delete_delete
  	::
  	unique_groups:
  		dsid roleid
  	::
  	`)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  err = createOrUpdateTable(`
  	table: qf_permissions
  	fields:
  		dsid int required
  		roleid int required
  		permissions text required
  	::
  	foreign_keys:
  		dsid qf_document_structures on_delete_delete
  		roleid qf_roles on_delete_delete
  	::
  	unique_groups:
  		dsid roleid
  	::
  	`)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  err = createOrUpdateTable(`
  	table: qf_user_roles
  	fields:
  		userid int required
  		roleid int required
  	::
  	foreign_keys:
  		userid users on_delete_delete
  		roleid qf_roles on_delete_delete
  	::
  	unique_groups:
  		userid roleid
  	::
  	`)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  err = createOrUpdateTable(`
  	table: qf_buttons
  	fields:
  		name string required
  		dsid int required
  		url_prefix string required
  	::
  	foreign_keys:
  		dsid qf_document_structures on_delete_delete
  	::
  	unique_groups:
  		name dsid
  	::
  	`)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  err = createOrUpdateTable(`
  	table: qf_mylistoptions
  	fields:
  		userid int required
  		dsid int required
  		field string required
  		data string required
  	::
  	foreign_keys:
  		userid users on_delete_delete
  		dsid qf_document_structures on_delete_delete
  	::
  	`)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  err = createOrUpdateTable(`
  	table: qf_files_for_delete
  	fields:
  		created_by int required
  		filepath string required
  	::
  	`)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  err = createOrUpdateTable(`
  	table: qf_btns_and_roles
  	fields:
  		roleid int required
  		buttonid int required
  	::
  	foreign_keys:
  		roleid qf_roles on_delete_delete
  		buttonid qf_buttons on_delete_delete
  	::
  	unique_groups:
  		roleid buttonid
  	::
  	`)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  fmt.Fprintf(w, "Setup Completed.")
}


func AddFORMS814Handlers(r *mux.Router) {

  // Please don't change the paths.

  // Please call this link first to do your setup.
  r.HandleFunc("/forms814-setup/", forms814Setup)

  // // admin pages
  // r.HandleFunc("/qf-page/", qfPage)
  // r.HandleFunc("/qf-upgrade/", qfUpgrade)

  // document structure links
  r.HandleFunc("/new-document-structure/", newDocumentStructure)
  r.HandleFunc("/list-document-structures/", listDocumentStructures)
  r.HandleFunc("/delete-document-structure/{document-structure}/", deleteDocumentStructure)
  r.HandleFunc("/view-document-structure/{document-structure}/", viewDocumentStructure)
  r.HandleFunc("/edit-document-structure-permissions/{document-structure}/", editDocumentStructurePermissions)
  // r.HandleFunc("/edit-document-structure/{document-structure}/", editDocumentStructure)
  // r.HandleFunc("/update-document-structure-name/{document-structure}/", updateDocumentStructureName)
  // r.HandleFunc("/update-help-text/{document-structure}/", updateHelpText)
  // r.HandleFunc("/update-field-labels/{document-structure}/", updateFieldLabels)
  // r.HandleFunc("/delete-fields/{document-structure}/", deleteFields)
  // r.HandleFunc("/change-fields-order/{document-structure}/", changeFieldsOrder)
  // r.HandleFunc("/add-fields/{document-structure}/", addFields)
  r.HandleFunc("/new-ds-from-template/{document-structure}/", newDSFromTemplate)


  // publicity document structure links
  r.HandleFunc("/make-public/{document-structure}/", makePublic)
  r.HandleFunc("/undo-make-public/{document-structure}/", undoMakePublic)


  // roles links
  r.HandleFunc("/roles-view/", rolesView)
  r.HandleFunc("/new-roles/", newRole)
  r.HandleFunc("/rename-role/", renameRole)
  r.HandleFunc("/delete-role/{role}/", deleteRole)
  r.HandleFunc("/users-to-roles-list/", usersToRolesList)
  r.HandleFunc("/users-to-roles-list/{page:[0-9]+}/", usersToRolesList)
  r.HandleFunc("/edit-user-roles/{userid}/", editUserRoles)
  r.HandleFunc("/remove-role-from-user/{userid}/{role}/", removeRoleFromUser)
  r.HandleFunc("/delete-role-permissions/{document-structure}/{role}/", deleteRolePermissions)
  r.HandleFunc("/user-details/{userid}/", userDetails)
  r.HandleFunc("/view-role-members/{role}/", viewRoleMembers)
  r.HandleFunc("/remove-role-from-user-2/{userid}/{role}/", removeRoleFromUser2)

  // // document links
  // r.HandleFunc("/create/{document-structure}/", createDocument)
  // r.HandleFunc("/update/{document-structure}/{id:[0-9]+}/", updateDocument)
  // r.HandleFunc("/list/{document-structure}/", listDocuments)
  // r.HandleFunc("/list/{document-structure}/{page:[0-9]+}/", listDocuments)
  // r.HandleFunc("/delete/{document-structure}/{id:[0-9]+}/", deleteDocument)
  // r.HandleFunc("/search/{document-structure}/", searchDocuments)
  // r.HandleFunc("/search-results/{document-structure}/", searchResults)
  // r.HandleFunc("/search-results/{document-structure}/{page:[0-9]+}/", searchResults)
  // r.HandleFunc("/delete-search-results/{document-structure}/", deleteSearchResults)
  // r.HandleFunc("/date-lists/{document-structure}/", dateLists)
  // r.HandleFunc("/date-lists/{document-structure}/{page:[0-9]+}/", dateLists)
  // r.HandleFunc("/date-list/{document-structure}/{date}/", dateList)
  // r.HandleFunc("/date-list/{document-structure}/{date}/{page:[0-9]+}/", dateList)
  // r.HandleFunc("/approved-list/{document-structure}/", approvedList)
  // r.HandleFunc("/approved-list/{document-structure}/{page:[0-9]+}/", approvedList)
  // r.HandleFunc("/unapproved-list/{document-structure}/", unapprovedList)
  // r.HandleFunc("/unapproved-list/{document-structure}/{page:[0-9]+}/", unapprovedList)

  // file links
  r.HandleFunc("/serve-js/{library}/", serveJS)
  r.HandleFunc("/qf-file/", serveFileForQF)
  r.HandleFunc("/delete-file/{document-structure}/{id:[0-9]+}/{name}/", deleteFile)
  r.HandleFunc("/complete-files-delete/", completeFilesDelete)
  r.HandleFunc("/delete-file-from-browser/", deleteFileFromBrowser)

  // // My List links
  // r.HandleFunc("/mylist-setup/{document-structure}/", myListSetup)
  // r.HandleFunc("/remove-list-config/{document-structure}/{field}/{data}/", removeOneMylistConfig)
  // r.HandleFunc("/mylist/{document-structure}/", myList)
  // r.HandleFunc("/mylist/{document-structure}/{page:[0-9]+}/", myList)

  // // Buttons
  // r.HandleFunc("/create-button/", createButton)
  // r.HandleFunc("/list-buttons/", listButtons)
  // r.HandleFunc("/delete-button/{id}/", deleteButton)

}

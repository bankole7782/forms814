package forms814

import (
	"net/http"
	"github.com/gorilla/mux"
	"fmt"
)


func makePublic(w http.ResponseWriter, r *http.Request) {
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

  err = FRCL.UpdateRowsAny(fmt.Sprintf(`
    table: qf_document_structures
    where:
      fullname = '%s'
    `, ds), map[string]interface{} { "public": true})
  if err != nil {
  	errorPage(w, err.Error())
  	return
  }
  
  http.Redirect(w, r, fmt.Sprintf("/view-document-structure/%s/", ds), 307)
}


func undoMakePublic(w http.ResponseWriter, r *http.Request) {
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

  err = FRCL.UpdateRowsAny(fmt.Sprintf(`
    table: qf_document_structures
    where:
      fullname = '%s'
    `, ds), map[string]interface{} { "public": false})
  if err != nil {
  	errorPage(w, err.Error())
  	return
  }
  
  http.Redirect(w, r, fmt.Sprintf("/view-document-structure/%s/", ds), 307)
}


func publicState(documentStructure string) (bool, error) {
  row, err := FRCL.SearchForOne(fmt.Sprintf(`
    table: qf_document_structures
    where:
      fullname = '%s'
    `, documentStructure))
  if err != nil {
    return false, err
  }

  return (*row)["public"].(bool), nil
}

package main

import (
  "net/http"
  "github.com/bankole7782/flaarum"
  "github.com/bankole7782/forms814"
  "os"
  "github.com/gorilla/mux"
  "strconv"
)


func main() {
	addr := os.Getenv("F8_FLAARUM_ADDR")
  keyStr := os.Getenv("F8_FLAARUM_KEYSTR")
  projName := os.Getenv("F8_FLAARUM_PROJ")

  cl := flaarum.NewClient(addr, keyStr, projName)
	if err := cl.Ping(); err != nil {
    panic(err)
  }

  // FORMS814 setup. Very important
  forms814.FRCL = cl

  forms814.Admins = []int64{1, }
  forms814.Inspectors = []int64{5,}

  // This sample makes use of environment variables to get the current user. Real life application
  // could save a random string to the browser cookies. And this random string point to a userid
  // in the database.
  // The function accepts http.Request as argument which can be used to get the cookies.
  forms814.GetCurrentUser = func(r *http.Request) (int64, error) {
    userid := os.Getenv("USERID")
    if userid == "" {
      return 0, nil
    }
    useridInt64, err := strconv.ParseInt(userid, 10, 64)
    if err != nil {
      return 0, err
    }
    return useridInt64, nil
  }

  // forms814.ExtraCodeMap[1] = qf.ExtraCode{CanCreateFn: testCreateFn}

  forms814.BucketName = os.Getenv("F8_GCLOUD_BUCKET")

  // forms814.BaseTemplate = "basetemplate.html"
  r := mux.NewRouter()
  forms814.AddHandlers(r)

  http.ListenAndServe(":3001", r)
}

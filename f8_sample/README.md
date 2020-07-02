# f8_sample
An example use of forms814.


## How to Run
Set the following environment variables before running this project:

1.	`F8_FLAARUM_ADDR` to the address of the flaarum server.
2.	`F8_FLAARUM_KEYSTR` to the key string for the flaarum server.
3.	`F8_FLAARUM_PROJ` to the name of the project you want to run flaarum on. Use `first_proj` for the default project.

Copy the `f8_files` from the [forms814](https://github.com/bankole7782/forms814) to the root folder of your
copy of this repository.

This test project is designed to be minimal. So it uses environment variable to get the current
user while the forms814 project provides facilities to get the current user using cookies.

To run this project do it this way `USERID=3 go run main.go` with the current user being 3 for example.
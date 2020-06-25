# How it Works.


## Viewing Documents and Listing Documents

It makes use of the theory that every piece of data can be stored in a string.


## UI

This project makes use of jquery for things like adding repetitions of some fields. This
project is impossible without the use of javascript which jquery makes a lot easier.


## Document Structures.

On creation of a document structure, its data is first saved to a table `f8_document_structures`
and `f8_fields` before creating a table for the document structure. This data is used to create forms.
Forms such as new document form and the update document form.


## Editing Tables

For comfort sake and for the fact that the primary keys of tables are not used in the UI,
every editing of tables is programmed to be a delete of old information and insertion of
new information.

So for every edit action, the primary keys of the child tables would advance.


## Files

Any uploaded file is not stored directly on database. Instead it is stored on Google Cloud Storage (GCS).

For conflicts sake eg. Five or more people uploading documents all with the name certificate.jpg, the names
are replaced with an long random string. The long random string reduces the case of conflicts.

What is stored about files to the database is the path of the file. This part consists of the database name
of the table (this enables renaming and categorization) followed by a forward slash and ends
with the generated random name for the document.

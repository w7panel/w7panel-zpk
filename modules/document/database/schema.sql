CREATE TABLE ims_document_library (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 namespace varchar(96) NOT NULL DEFAULT 'app_doc',
 identifie varchar(96) NOT NULL,
 name varchar(255) NOT NULL,
 description text NOT NULL DEFAULT '',
 user_id integer NOT NULL,
 username varchar(255) NOT NULL DEFAULT '',
 public integer NOT NULL DEFAULT 0,
 protected integer NOT NULL DEFAULT 0,
 deleted_at datetime,
 created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
 updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX document_library_ref ON ims_document_library(namespace,identifie);
CREATE INDEX document_library_owner ON ims_document_library(user_id);
CREATE INDEX document_library_deleted_at ON ims_document_library(deleted_at);
CREATE TABLE ims_document_file (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 library_id integer NOT NULL,
 path varchar(255) NOT NULL,
 title varchar(255) NOT NULL,
 storage_key varchar(255) NOT NULL,
 size bigint NOT NULL,
 position integer NOT NULL DEFAULT 0,
 protected integer NOT NULL DEFAULT 0,
 deleted_at datetime,
 updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX document_file_path ON ims_document_file(library_id,path);
CREATE INDEX document_file_deleted_at ON ims_document_file(deleted_at);

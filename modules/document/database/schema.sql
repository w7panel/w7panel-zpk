CREATE TABLE ims_document_category (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 identifie varchar(96) NOT NULL,
 name varchar(255) NOT NULL,
 description text NOT NULL DEFAULT '',
 user_id integer NOT NULL,
 deleted_at datetime,
 created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
 updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX document_category_identifie ON ims_document_category(identifie);
CREATE INDEX document_category_deleted_at ON ims_document_category(deleted_at);
CREATE TABLE ims_document_library (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 category varchar(96) NOT NULL,
 identifie varchar(96) NOT NULL,
 name varchar(255) NOT NULL,
 description text NOT NULL DEFAULT '',
 user_id integer NOT NULL,
 public integer NOT NULL DEFAULT 0,
 protected integer NOT NULL DEFAULT 0,
 deleted_at datetime,
 created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
 updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX document_library_ref ON ims_document_library(category,identifie);
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
CREATE TABLE ims_document_session (
 id varchar(64) PRIMARY KEY NOT NULL,
 user_info text NOT NULL,
 expires_at datetime NOT NULL,
 deleted_at datetime,
 created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
 updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX document_session_expires_at ON ims_document_session(expires_at);
CREATE INDEX document_session_deleted_at ON ims_document_session(deleted_at);

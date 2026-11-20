CREATE TABLE ims_document_category (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 identifie varchar(96) NOT NULL,
 name varchar(255) NOT NULL,
 description text NOT NULL DEFAULT '',
 user_id integer NOT NULL,
 revision integer NOT NULL DEFAULT 1,
 created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
 updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX document_category_identifie ON ims_document_category(identifie);
CREATE TABLE ims_document_library (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 category varchar(96) NOT NULL,
 identifie varchar(96) NOT NULL,
 name varchar(255) NOT NULL,
 description text NOT NULL DEFAULT '',
 user_id integer NOT NULL,
 revision integer NOT NULL DEFAULT 1,
 ready integer NOT NULL DEFAULT 0,
 created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
 updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX document_library_ref ON ims_document_library(category,identifie);
CREATE INDEX document_library_owner ON ims_document_library(user_id);
CREATE TABLE ims_document_file (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 library_id integer NOT NULL,
 path varchar(255) NOT NULL,
 title varchar(255) NOT NULL,
 storage_key varchar(255) NOT NULL,
 size bigint NOT NULL,
 position integer NOT NULL DEFAULT 0,
 revision integer NOT NULL DEFAULT 1,
 updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX document_file_path ON ims_document_file(library_id,path);
CREATE TABLE ims_document_download_link (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 library_id integer NOT NULL,
 token_hash varchar(64) NOT NULL,
 paths text NOT NULL,
 user_id integer NOT NULL,
 expires_at datetime NOT NULL,
 revoked integer NOT NULL DEFAULT 0,
 created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX document_download_link_hash ON ims_document_download_link(token_hash);

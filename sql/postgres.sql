drop table if exists tags;
drop table if exists article_tags;
drop table if exists pubkeys cascade;
drop table if exists users cascade;
drop table if exists articles cascade;
drop table if exists comments;

create table tags (
	id serial unique,
	created timestamp without time zone default now(),
	tag text
);

create table article_tags (
	articleid int,
	tagid int
);

create table users (
	id serial unique,
	created timestamp without time zone default now(),
	fname text,
	lname text,
	email text,
	hash text,
	username text unique not null
);

create table pubkeys (
	id serial unique,
	created timestamp without time zone default now(),
	userid int references users (id) on delete cascade,
	key text
);

create table articles (
	id serial unique,
	slug text not null,
	created timestamp without time zone default now(),
	edited timestamp without time zone default now(),
	published timestamp without time zone default now(),
	live bool default false,
	authorid int references users (id),
	title text not null,
	body text not null,
	tsv tsvector,
	sig text
);

create index articles_ts_idx on articles using gin (tsv);
create index articles_title_trgm_idx ON articles using gin (title gin_trgm_ops);
create index articles_body_trgm_idx ON articles using gin (body gin_trgm_ops);

CREATE or replace FUNCTION article_slug_trigger() RETURNS trigger AS $$
begin
  new.slug :=
      -- wait to replace the space so we can get readable slugs
      lower(regexp_replace(regexp_replace(new.title, '[^a-zA-Z0-9 -]', '', 'g'), '\s', '-', 'g'));
  return new;
end
$$ LANGUAGE plpgsql;

CREATE TRIGGER articlesligify BEFORE INSERT OR UPDATE
    ON articles FOR EACH ROW EXECUTE PROCEDURE article_slug_trigger();

CREATE or replace FUNCTION articles_ts_trigger() RETURNS trigger AS $$
begin
  new.tsv :=
     setweight(to_tsvector('pg_catalog.english', coalesce(new.title,'')), 'A') ||
     setweight(to_tsvector('pg_catalog.english', coalesce(new.body,'')), 'B');
  return new;
end
$$ LANGUAGE plpgsql;

CREATE TRIGGER tsvectorupdate BEFORE INSERT OR UPDATE
    ON articles FOR EACH ROW EXECUTE PROCEDURE articles_ts_trigger();


create table comments (
	id serial unique,
	created timestamp without time zone default now(),
	pid int default 0 references comments (id) on delete set default ,
	pkid int references pubkeys (id),
	userid int references users (id) on delete cascade,
	comment text,
	sig text
);

create or replace function hash(pass text) returns text as $$
	select crypt(pass, gen_salt('bf', 10));	
$$ language sql;

insert into users (fname, lname, username, hash) values ('Charlie', 'Root', 'root', hash('omgSnakes'));
insert into pubkeys (userid, key) values (1, 'untrusted comment: signify public key
RWSYzBxZQY5obtJcBPKBQHzy6EpyV/D5VpDB58f1Hrn4NqaC1Jo2fSz9');
insert into tags (tag) values ('OpenBSD');
insert into tags (tag) values ('FreeBSD');
insert into tags (tag) values ('NetBSD');
insert into tags (tag) values ('HardenedBSD');
insert into tags (tag) values ('DragonflyBSD');

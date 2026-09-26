CREATE VIRTUAL TABLE media_items_search_index USING fts5(
    title,
    description,
    tokenize=porter
  );
-- +statement
CREATE TRIGGER media_items_search_index_insert AFTER INSERT ON media_items BEGIN
    INSERT INTO media_items_search_index(
      rowid,
      title,
      description
    )
    VALUES(
      new.id,
      new.title,
      new.description
    );
  END;
-- +statement
CREATE TRIGGER media_items_search_index_update AFTER UPDATE ON media_items BEGIN
    UPDATE media_items_search_index SET
      title = new.title,
      description = new.description
    WHERE
      rowid = old.id;
  END;
-- +statement
CREATE TRIGGER media_items_search_index_delete AFTER DELETE ON media_items BEGIN
    DELETE FROM media_items_search_index WHERE rowid = old.id;
  END;
-- +statement

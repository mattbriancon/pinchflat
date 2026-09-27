CREATE UNIQUE INDEX sources_collection_id_media_profile_id_title_filter_regex_index ON sources (
    collection_id,
    media_profile_id,
    IFNULL(title_filter_regex, '')
  );
-- +statement

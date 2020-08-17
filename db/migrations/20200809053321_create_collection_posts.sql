-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE `CollectionPosts` (
  `CollectionId` varchar(26) NOT NULL,
  `PostId` varchar(26) NOT NULL,
  PRIMARY KEY (`CollectionId`,`PostId`),
  KEY `idx_collectionposts_collection_id` (`CollectionId`),
  KEY `idx_collectionposts_post_id` (`PostId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS `CollectionPosts`;

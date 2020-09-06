-- +goose Up
-- SQL in this section is executed when the migration is applied.
-- TODO: チームに紐づくテーブルを見るにはそのteamのメンバーである事が必要。
-- すでに脱退済みなら見れない、他も同様。
-- 一度脱退し、再度joinすると以前と同じ様に見れる仕様。
CREATE TABLE `Votes` (
  `PostId` varchar(26) NOT NULL,
  `UserId` varchar(26) NOT NULL,
  `Type` varchar(26) NOT NULL,
  `Tags` text,
  `TeamId` varchar(26) DEFAULT NULL,
  `FirstPostRev` int(11) DEFAULT NULL,
  `LastPostRev` int(11) DEFAULT NULL,
  `CreateAt` bigint(20) DEFAULT NULL,
  `InvalidateAt` bigint(20) DEFAULT NULL,
  `CompletedAt` bigint(20) DEFAULT NULL,
  `CompletedBy` varchar(26) DEFAULT NULL,
  `RejectedAt` bigint(20) DEFAULT NULL,
  `RejectedBy` varchar(26) DEFAULT NULL,
  PRIMARY KEY (`UserId`,`Type`,`PostId`),
  KEY `idx_votes_user_id_type_create_at` (`UserId`,`Type`,`CreateAt`),
  FULLTEXT KEY `idx_votes_tags_txt` (`Tags`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS `Votes`;

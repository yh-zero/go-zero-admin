-- Correct only the untouched historical HTTP-method seed: POST and GET were
-- both assigned value=2. The intended sequence is POST=1, GET=2, PUT=3, DELETE=4.
-- Exact IDs, original timestamps and all seed business fields must still match.
-- Custom/edited rows or an occupied value=1 are left untouched for manual review.
-- A fresh installation already seeds POST=1, so this migration does nothing.
-- The migration runner backs up the selected database before applying changes.
SET NAMES utf8mb4 COLLATE utf8mb4_general_ci;
START TRANSACTION;

UPDATE sys_dictionary_info AS post_seed
JOIN sys_dictionary_info AS get_seed ON get_seed.id=4
JOIN sys_dictionaries AS dictionary_seed ON dictionary_seed.id=2
LEFT JOIN sys_dictionary_info AS occupied
  ON occupied.sys_dictionary_id=2 AND occupied.value=1 AND occupied.deleted_at IS NULL
SET post_seed.value=1
WHERE post_seed.id=3
  AND post_seed.sys_dictionary_id=2 AND post_seed.deleted_at IS NULL
  AND CAST(post_seed.label AS BINARY)='POST' AND post_seed.value=2 AND CAST(post_seed.extend AS BINARY)='POST'
  AND post_seed.status=1 AND post_seed.sort=0
  AND post_seed.created_at='2024-01-16 17:38:19.939'
  AND post_seed.updated_at='2024-01-18 17:26:35.798'
  AND get_seed.sys_dictionary_id=2 AND get_seed.deleted_at IS NULL
  AND CAST(get_seed.label AS BINARY)='GET' AND get_seed.value=2 AND CAST(get_seed.extend AS BINARY)='GET'
  AND get_seed.status=1 AND get_seed.sort=0
  AND get_seed.created_at='2024-01-16 17:38:36.624'
  AND get_seed.updated_at='2024-01-16 17:38:36.624'
  AND dictionary_seed.deleted_at IS NULL
  AND CAST(dictionary_seed.name AS BINARY)='api请求' AND CAST(dictionary_seed.type AS BINARY)='method'
  AND dictionary_seed.status=1 AND CAST(dictionary_seed.`desc` AS BINARY)='api请求'
  AND dictionary_seed.created_at='2024-01-16 17:38:10.566'
  AND dictionary_seed.updated_at='2024-01-16 17:38:10.566'
  AND occupied.id IS NULL;

COMMIT;

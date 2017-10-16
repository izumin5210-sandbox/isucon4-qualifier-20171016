CREATE TABLE IF NOT EXISTS `users` (
  `id` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `login` varchar(255) NOT NULL UNIQUE,
  `password_hash` varchar(255) NOT NULL,
  `salt` varchar(255) NOT NULL
) DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `login_log` (
  `id` bigint NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `created_at` datetime NOT NULL,
  `user_id` int,
  `login` varchar(255) NOT NULL,
  `ip` varchar(255) NOT NULL,
  `succeeded` tinyint NOT NULL
) DEFAULT CHARSET=utf8;

CREATE INDEX
  index_login_log_on_user_id_and_id
ON login_log (
  user_id,
  id
);

CREATE INDEX
  index_login_log_on_user_id_and_succeeded
ON login_log (
  user_id,
  succeeded
);

CREATE INDEX
  index_login_log_on_ip_and_id
ON login_log (
  ip,
  id
);

CREATE INDEX
  index_users_on_login
ON users (
  login
);

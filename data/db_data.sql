create table if not exists `tb_porn_index`
(
    `id`         int primary key auto_increment,
    `en_name`    varchar(255) not null,
    `jp_name`    varchar(255),
    `episode`    varchar(64)  not null,
    `image_path` varchar(255),
    `video_path` varchar(255) not null
) engine = innodb
  default character set utf8;
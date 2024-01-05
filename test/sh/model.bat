@REM 项目根目录  运行例子： .\pkg\test\model.bat user user
@REM  第一个参数的数据库名  第二个参数是表名
set baseName=%1
set tableName=%2
set mysql=root:123456@tcp(127.0.0.1:3306)/beyond_goZero_%baseName%


goctl model mysql datasource --dir application/%baseName%/rpc/internal/model --table %tableName% --cache true --url=%mysql%

@REM goctl model mysql datasource --dir application/%baseName%/mq/internal/model --table %tableName% --url=%mysql%






@REM goctl model mysql datasource -url="%mysql%" -table="%table%" -c -dir %dir% -home ./dev/goctl
@REM goctl model mysql datasource --dir ./internal/model --table user --cache true --url "root:Zsg123456@tcp(127.0.0.1:3306)/beyond_user"
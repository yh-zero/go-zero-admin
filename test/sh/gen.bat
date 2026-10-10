@echo off
REM ============================================
REM go-zero-admin 代码生成工具
REM 项目根目录运行
REM ============================================

if "%1"=="" goto :usage
if "%1"=="api" goto :api
if "%1"=="rpc" goto :rpc
if "%1"=="model" goto :model
goto :usage

:api
REM 生成 API 代码
REM 用法: gen.bat api applet
if "%2"=="" goto :usage_api
if not exist test\goctl\api\handler.tpl (
    echo Error: Custom template not found: test\goctl\api\handler.tpl
    exit /b 1
)
if not exist test\goctl\api\main.tpl (
    echo Error: Custom template not found: test\goctl\api\main.tpl
    exit /b 1
)
echo Generating API code for %2...
goctl api go -api application\%2\api\desc\%2.api -dir application\%2\api\ -home test\goctl\ -style=go_zero
goto :eof

:rpc
REM 生成 RPC 代码
REM 用法: gen.bat rpc applet applet
if "%2"=="" goto :usage_rpc
if "%3"=="" goto :usage_rpc
echo Generating RPC code for %2/%3...
goctl rpc protoc application\%2\rpc\desc\%3.proto --go_out=application\%2\rpc --go-grpc_out=application\%2\rpc --zrpc_out=application\%2\rpc\ -m --style=go_zero --name-from-filename
goto :eof

:model
REM 生成 Model 代码
REM 用法: gen.bat model ^<数据库名^> ^<表名^>
if "%2"=="" goto :usage_model
if "%3"=="" goto :usage_model
echo Generating Model code for %2.%3...
goctl model mysql datasource --dir application/%2/rpc/internal/model --table %3 --cache true --url=root:123456@tcp(127.0.0.1:3306)/goZero-admin
goto :eof

:usage
echo Usage: gen.bat ^<command^> [args...]
echo.
echo Commands:
echo   api ^<name^>           Generate API code (e.g., gen.bat api applet)
echo   rpc ^<name^> ^<rpc^>     Generate RPC code (e.g., gen.bat rpc applet applet)
echo   model ^<db^> ^<table^>   Generate Model code (e.g., gen.bat model user sys_users)
exit /b 1

:usage_api
echo Usage: gen.bat api ^<name^>
echo Example: gen.bat api applet
exit /b 1

:usage_rpc
echo Usage: gen.bat rpc ^<name^> ^<rpc^>
echo Example: gen.bat rpc applet applet
exit /b 1

:usage_model
echo Usage: gen.bat model ^<db^> ^<table^>
echo Example: gen.bat model user sys_users
exit /b 1

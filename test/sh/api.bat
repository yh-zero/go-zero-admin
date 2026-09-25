@REM 注意 运用了自定义模板

@REM 项目根目录  运行例子1： .\test\sh\api.bat applet

set name=%1

@REM Use the version-independent API overrides; otherwise goctl silently uses defaults.
if not exist test\goctl\api\handler.tpl exit /b 1
if not exist test\goctl\api\main.tpl exit /b 1
goctl api go -api application\%name%\api\desc\%name%.api -dir application\%name%\api\ -home test\goctl\ -style=go_zero


@REM if "%api%" == "api" (
@REM    goctl api go -api application\%dir%\api\%dir%.api -dir application\%dir%\api\
@REM ) else (
@REM    goctl api go -api application\%dir%\%dir%.api -dir application\%dir%\
@REM )



@REM goctl api go -api application\%dir%\%api% -dir %dir% -home ./dev/goctl
@REM goctl api go --dir=./ --api applet.api

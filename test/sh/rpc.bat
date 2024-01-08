@REM 项目根目录  运行例子： .\test\sh\rpc.bat applet  applet


set name=%1
set rpc=%2

goctl rpc protoc application\%name%\rpc\desc\%rpc%.proto --go_out=application\%name%\rpc --go-grpc_out=application\%name%\rpc --zrpc_out=application\%name%\rpc\ -m --style=go_zero

@REM goctl rpc protoc application\usre\rpc\usre.proto --go_out=application\usre\rpc --go-grpc_out=application\usre\rpc --zrpc_out=application\usre\rpc\


@REM goctl rpc protoc ./user.proto --go_out=. --go-grpc_out=. --zrpc_out=./

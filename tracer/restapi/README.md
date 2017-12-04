# REST API specification
The API specifications defined at `app.yaml`.
See [app.yaml](./app.yaml).

## How to show specification via Swagger UI
1. Execute the `./swagger-ui.sh` on terminal.
   Swagger UI server will start on foreground.
2. Open [http://localhost:8080/](http://localhost:8080/) on web browser.
3. Press `Ctrl+C` on terminal to stop the Swagger UI server.


## GET /api/v1.0/log/{LOG_ID}/search
`min-id`と`max-id`、もしくは`min-timestamp`と`max-timestamp`が指定されなかった場合、全件スキャンが発生するためレスポンスが遅くなる。
可能な限り`id`か`timestamp`を指定すること。

# Status API
Statusは、フラットはKey/Valueの辞書と、status versionを返す。
status versionとは0から始まる整数で、statusが更新されるたびにstatus versionが加算される。

statusを更新するには、更新対象のstatus versionと更新対象のKey/ValueのペアをPOSTする。
もしstatus versionが異なっているなら、"409 Conflict"と最新のstatusとstatus versionを返し、更新には失敗する。
更新に成功した場合、"200 OK"と新しいstatus versionを返す。

# Watch API
Statusの変化を通知するLong poling用のAPI。
versionとtimeoutの2つを引数に受け取る。
下記のいずれかの状態を満たしたときに、新しいStatusVersionと、statusを返す。
* StatusVersionが変化した
* リクエスト受付からTIMEOUT秒経過


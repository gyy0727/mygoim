package dao

//*Redis 操作命令及逻辑说明：

//?1. pingRedis: 发送 `SET PING "PONG"`，检查 Redis 连接是否正常。

//?2. AddMapping: 
//*   - 若 `mid > 0`，执行 `HSET mid_<mid> <key> <server>` 和 `EXPIRE mid_<mid> <expire>`。
//*   - 执行 `SET key_<key> <server>` 和 `EXPIRE key_<key> <expire>`，存储映射关系并设置过期时间。

//?3. ExpireMapping: 
//*   - 若 `mid > 0`，执行 `EXPIRE mid_<mid> <expire>`。
//*   - 执行 `EXPIRE key_<key> <expire>`，更新映射关系的过期时间。

//?4. DelMapping: 
//*   - 若 `mid > 0`，执行 `HDEL mid_<mid> <key>`。
//*   - 执行 `DEL key_<key>`，删除映射关系。

//?5. ServersByKeys: 执行 `MGET key_<key1> key_<key2> ...`，批量获取与 `keys` 对应的值。

//?6. KeysByMids: 执行 `HGETALL mid_<mid1> mid_<mid2> ...`，批量获取与 `mids` 对应的哈希数据。

//?7. AddServerOnline: 
//*   - 将 `online` 数据序列化为 JSON。
//*   - 执行 `HSET ol_<server> <hashKey> <onlineData>` 和 `EXPIRE ol_<server> <expire>`，存储在线状态并设置过期时间。

//?8. ServerOnline: 执行 `HGET ol_<server> <hashKey>`，获取在线状态并反序列化为 `model.Online`。

//?9. DelServerOnline: 执行 `DEL ol_<server>`，删除服务器的在线状态。

//*总结：主要使用 `SET`、`GET`、`MGET`、`HSET`、`HGET`、`HGETALL`、`HDEL`、`EXPIRE`、`DEL` 等命令，
//*用于管理用户与服务器的映射关系及服务器的在线状态。

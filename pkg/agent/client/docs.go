package client

/*

此包用于将 agent 中的 client.Client 转换为rest的api。
在service端，实现了另一个 client.Client 接口，其后端为agent的该rest api。

目的在于在service实现一个 "原生” 的资源操作client 以操作agent所在集群的资源。

这个 rest api 仅对内提供服务。
*/

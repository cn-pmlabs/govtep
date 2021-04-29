[toc]

# SYSTEM DESIGN

![VTEP](https://note.youdao.com/yws/public/resource/22f28b35e99be7a3db88b4285d374522/xmlnote/BBA74FEE74C34854A8C08B065054418D/16272)



VTEP设备上维护OVSDB数据库，VXLAN相关配置以表项的形式保存在该数据库中。控制器与VTEP设备上的OVSDB服务器建立连接，二者采用OVSDB控制协议进行交互并操作OVSDB数据库中的数据。OVSDB VTEP服务从OVSDB服务器获取数据库中的数据，将其转变为VXLAN相关配置（例如创建或删除VXLAN、创建或删除VXLAN隧道）下发到设备上。同时，OVSDB VTEP服务也会通过OVSDB服务器，将本地的用户侧接入端口和VXLAN隧道全局源地址信息添加到数据库中，并上报给控制器。

用户可以通过命令行和控制器同时控制交换机，但是使能OVSDB后，vxlan相关功能及相关接口建议仅通过控制器进行配置

表名 | 描述 | 备注 |
Physical_Locator_Set | 

### ovn


### db


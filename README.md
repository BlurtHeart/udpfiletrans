# 简单文件传输

## Synopsis

通过UDP来在两台服务器之间传输文件。

## 建立连接（握手）

1. 客户端发起一个读或写请求，请求内容包括：
    1. 读或写。
    2. 文件信息。文件名，文件大小，文件MD5等。

2. 服务端收到请求，响应一个ack包，如果是读请求，包内容包括：
    1. 读请求。
    2. 文件是否存在，是否有权限读取。如果不存在或没有权限，同志客户端，并直接退出此次请求。
    3. 如果请求中包含文件的大小以及MD5值，则根据文件大小计算服务器上的文件的MD5是否一致，如果一致，则响应文件ackSame，并返回文件的全部大小以及MD5值。否则，返回ackNSame。
    3. 文件信息。文件名，如果服务器存在此文件，则返回此文件的信息，文件大小，文件MD5值等。

3. 如果是写请求，包内容包括：
    1. 写请求。
    2. 文件是否存在，是否有权限写。如果不存在或没有权限，同志客户端，并直接退出此次请求。
    3. 如果文件存在，则根据文件大小计算服务器上的文件的MD5是否一致，如果一致，则响应文件ackSame，并返回文件的全部大小以及MD5值。否则，返回ackNSame。

4. 客户端收到响应之后，如果是读请求，响应ack code应该是ackSame，ackNSame，ackNPermit，ackNExist中的一种：
    1. ackSame。回一个确认包，包内包含文件数据的起始位置（断点续传）。
    2. ackNSame。回一个确认包，包内包含是否继续传输。
    3. ackNPermit。放弃传输，不回确认包。
    4. ackNExist。放弃传输，不回确认包。

5. 如果是写请求，响应ack code应该是ackSame，ackNSame，ackNPermit，ackNExist中的一种：
    1. ackSame。返回写成功。回一个确认包，包内包含文件数据的起始位置（断点续传）。
    2. ackNSame。根据服务端返回的文件大小和文件MD5值计算文件。回一个确认包，包内包含文件发送起始位置。
    3. ackNPermit。放弃传输，不回确认包。
    4. ackNExist。进入下一环节，开始发送数据。 

6. 服务器如果在等待响应。则根据以上具体响应内容决定是否继续传输以及传输开始位置。

## Disclaimer

Inspired by [tftp](https://github.com/pin/tftp)

tftp implemented by [pin](https://github.com/pin) is very good. 
I don't intend to modify it, but just to add some controls on UDP 
to assure file tranfer.

Document will given later...

Code is in progress, so it will continue evolving little by little 
and at this point I'm not really looking for contributions.

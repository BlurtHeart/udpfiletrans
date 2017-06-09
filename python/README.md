'''
* Autor: Steve Cloud
*
* Create on 2016/10/12
'''

# udpfiletrans
本项目旨在通过udp在两个不同的机器，甚至是网络之间可靠性的传输文件。

## changelog
使用msgpack代替json，是因为json会在序列化数据的时候decode解码，但是读utf8文件的时候会将字符切断，从而出现解码失败。</br>
没有使用cPickle的原因点击这里[Don't Pickle Your Data](http://www.benfrederickson.com/dont-pickle-your-data/)

## inner protocol
	{
		'filename':
		'file_md5'
		'file_path'
		'file_packets'
		'packet_md5'
		'packet_index'
		'file_offset'
		'data':''      # 块数据
    	'status'
	}

	status:
    	syn-ack         # 握手应答
    	exist			# file exist
    	block-ack       # 块收包正常
    	block-uncorrect # 块包不正常
    	complete        # 传输成功
    	failed          # 传输失败

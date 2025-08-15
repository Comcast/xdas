# XDAS
### **Overview**
This is XDAS implemented in Golang. XDAS is a **_high performant_ persistent** data cache/store service. It is designed with the following goals:
* Accessible via REST API
* Encrypt/decrypt data at rest
* Baseic data format transformation
* Large number of concurrent requests
* Low latency (ms latency)
* Live linear horizontal scalability
* Configurable retention for each record

Each record is identified by `keyspace` + `key`. Its value can be virtually anything. `keyspace` is case sensitive. `key` is NOT case sensitive.

### **API**
#### API endpoint: `/v2/<keyspace>/<key>?format=<format>`    
GET calls by default returns the value in format specified in the config. To override that, use the follow optional query parameter:
* format, optional, currently supported formats are:    
   not set (default) - return message as specified in config   
   json - return message in uncompressed json format   
   protobuf - return message in uncompress protobuf format   
   raw - return message in as it is stored in Redis  

PUT/POST calls use the following headers:    
* Content-Type, expected format for keyspace is set via config. Currently supported types are:   
   application/json (or json)   
   application/x-protobuf (or protobuf)    
   application/octet-stream for all other formats
* Content-Encoding, required if compression is used, currently supported formats are:   
    zstd   
    zlib   
    catch all "" (empty string)
* Xttl, optional, used to override config default (per keyspace) TTLs

DEL calls delete the key for that keyspace

#### Atomic Redis Operation API encdpoint: `/v2/inc/<keyspace>/<key>?n=<num>`
Keyspace that has atomicInc set to true in config can make PUT/POST call to atomic operation API. Value will atomically increment by the query parameter value n. If n is not present or 0, value will be incremented by 1. If n is negative, value is decremented.

Only POST and PUT are supported on this endpoint. Use `/v2/<keyspace>` for GET call. In order to support atomic operation by Redis, these keyspaces will not use protobuf and will not be encrypted.

#### Hashes keyspace endpoint: `/v2/<keyspace>/<key>`
Hahes keyspace also uses `field` query parameter. For GET method, the field is optional, if omitted, all elements under the key is return. For DEL method, it is required to prevent accidental deletion of the entire key.

Hashes based keyspace will automatically extend TTL on GET/PUT/POST.

Keyspace config `kind` must set to `hashes`

#### Multipart API encdpoint: `/v2/multi/<key>?ks=<keyspace1>,<keyspace2>...`    
GET calls return all the keyspaces set in the config. To override that, use the following optional query parameter:
* ks, optional, used to override config default

The output uses multipart standard. Content-Type will be set to `multipart/form-data; boundary=bd4...`. Either Content-length or "Transfer-Encoding: chunked" maybe returned. Each part is seperated by boundary and include the following headers:
* Content-Type, will always be present
* Content-Encoding, will only present if the content is compressed
* Namespace, will always be present, indicating the keyspace for the part

### **Encryption**
All string kind keyspaces stored at rest are encrypted (atomic and hashes are not) according to the config. Currently supported encryption is:
* Authenticated Encryption with Associated Data (AEAD) using GCM (AES-256)


### **Configuration**
See [config.json.template](configs/config.jsonc) for explaination.




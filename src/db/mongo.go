package db 


import(
	"gopkg.in/mgo.v2" 
	log "github.com/GameGophers/libs/nsq-logger" 
	"labix.org/v2/mgo/bson" 
	"os"
	"types"
	"time"
)


var _sess *mgo.Session

const(
	URL = ":27017"
	db_name = "game" 
	collection_name = "AUTH"
)

func init() {
	sess, err := mgo.Dial(URL)
	if err != nil {
		log.Error("mgo dial error",err,URL) 
		os.Exit(-1)
	}
	_sess = sess
}

//--------------------------------------------------------- get connection with db and collection 
// ! need close copy session
func C(collection string) (*mgo.Session, *mgo.Collection) { 
	// need close()!
	ms := _sess.Copy()
	
	c := ms.DB(db_name).C(collection) 
	return ms,c
}

//--------------------------------------------------------- is user exist
func IsExist(cert string,logintype int32) (int32,bool) {
	ms,c := C(collection_name)	
	defer ms.Close() 

	auth := &types.Auth{}
	err := c.Find(bson.M{"logintype":logintype,"cert":cert}).One(auth) 
	if err != nil {
		log.Debug("玩家不存在") 
		return 0,false
	}
	return auth.Id,true
}

//--------------------------------------------------------- create new user
func New(id int32,uuid,cert,host string,logintype int32) {
	ms,c := C(collection_name) 
	defer ms.Close()

	auth := &types.Auth{
		Id: 		id,
		Uuid: 		uuid,
		Cert: 		cert,
		Host: 		host,	
		LoginType: 	logintype,
		CreateTime: time.Now().Unix(),
	}

	err := c.Insert(auth)
	if err != nil {
		log.Error("create user auth error",err,auth) 
	}
}

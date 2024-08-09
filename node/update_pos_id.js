const { MongoClient } = require('mongodb');
const uri = "mongodb://skDBAdmin:sakthi%402022%24Pharma@localhost:27017/?authSource=admin&readPreference=primary&appname=MongoDB%20Compass&directConnection=true&ssl=false";
const client = new MongoClient(uri);
const CSVToJSON = require('csvtojson');
var dt = new Date()
var removeUTF8BOM = require('@stdlib/string-remove-utf8-bom');
var avail_products = []
var idx=0


async function main() {
    await client.connect();
    var idx = 0
    client.db("sakthi_dev").collection("code_product").find()
        .forEach(doc => {
            update(doc)
        })

}


async function update(r) {
    client.db("sakthi_dev").collection("rs_drug").updateOne({ "_id": r.pos_id },{$set:{avail_code_product:1}}).then(result => {
       if (result.modifiedCount==1) {
          console.log("Yes")
       }
    })
}

main().catch(console.error);

const { MongoClient } = require('mongodb');
const uri = "mongodb://skDBAdmin:sakthi%402022%24Pharma@localhost:27017/?authSource=admin&readPreference=primary&appname=MongoDB%20Compass&directConnection=true&ssl=false";
const client = new MongoClient(uri);
const CSVToJSON = require('csvtojson');
const axios = require('axios');
// const utf8 = require('utf8');
// var encoding = require("encoding");
//const {stripBom} =require('strip-bom');
var removeUTF8BOM = require('@stdlib/string-remove-utf8-bom');
var avail_products = []

function formatDate(date) {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');

    return year+''+month+''+day+'1000';
}

var dttime = formatDate(new Date());
console.log(dttime);

async function main() {
  await client.connect();
  var idx=0
  // api call
  axios.get('http://main.rsdrugs.in:8282/api/itmstk/dt/'+dttime).then(resp => {
    let data = resp.data
    // console.log('No of Rows found', data.length)
    data.forEach(r => {
      // console.log(r)
      update(r,idx++)
    })

  });
}





async function update(doc, idx) {
  // console.log(doc.ref_id)
  const p = await client.db("sakthi_live").collection("product").findOne({ pos_id: doc.ITM_CODE })
  if (p) {
    if (p.mrp != doc.MRP || p.stock != doc.AVAIL_QTY) {
      insertDoc = {
        $set: {
          mrp: doc.MRP,
          stock: doc.AVAIL_QTY,
          updated_on: new Date()
        }
      }
      //FIND THE PRODUCT CODE , PRICE,,,,,FROM THE PRODUCT
      // console.log(ud)
      try {
        client.db("sakthi_live").collection("product").updateOne({ pos_id: doc.ITM_CODE }, insertDoc).then(res => {
          console.log(res, idx)
        })
      }
      catch (e) {
        console.log(e, idx)
      }
    }
  }
}
main().catch(console.error);
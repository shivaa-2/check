const { MongoClient } = require('mongodb');
const uri = "mongodb://skDBAdmin:sakthi%402022%24Pharma@localhost:27017/?authSource=admin&readPreference=primary&appname=MongoDB%20Compass&directConnection=true&ssl=false";
const client = new MongoClient(uri);
const CSVToJSON = require('csvtojson');
var dt = new Date()
var removeUTF8BOM = require('@stdlib/string-remove-utf8-bom');
var avail_products = []


async function main() {
    // avail_products = await CSVToJSON().fromFile('code_products.csv')
    //     .then(products => {
    //         return products
    //     })
    //console.log(avail_products)

    await client.connect();
    var idx = 0
    client.db("sakthi_dev").collection("code_products").find()
        .forEach(doc => {
            //console.log(doc._id, doc.medicine_name, doc.medicine_name.replace(/ /gm, '-').replace(/\'/gm, '-').replace(/\./gm, '-').toLowerCase())
            update(doc)
        })

}


async function update(r) {
    client.db("sakthi_dev").collection("new_product").findOne({ "doc.id": r._id }).then(doc => {
        if (doc && doc != null) {
            const idoc = {
                manufacturer: doc.doc.manufacturer_name,
                mrp: 0,
                hsnc: doc.doc.HSN_code,
                variant: doc.doc.Variant,
                about: doc.doc.about,
                cgst: 2.5,
                sgst: 2.5,
                discount: 20,
                stock: 100,
                category_id: doc.category.replace(/ /gm, '-'),
                category_name: doc.category,
                prescription_required: 'X',
                common_side_effect: doc.doc.commonSideEffect,
                serious_side_effect: doc.doc.seriousSideEffect,
                weight: doc.doc.weight,
                created_on: new Date(),
                updated_on: new Date()
            }
            try {
                client.db("sakthi_dev").collection("code_products").updateOne({ _id: r._id }, { $set: idoc }).then(res => {
                    console.log(res)
                })
            }
            catch (e) {
                console.log(e)
            }
        }
    })


    // const doc = r
    // let a = removeUTF8BOM(doc.About);
    // const docid = doc._id
    // a = a.replace(/b'/, '').replace(/\"/gm, '')
    //     .replace(/\/\/xc2/gm, '').replace(/\/\/xa0/gm, '')
    //     .replace(/\\x80/gm, '').replace(/\x99/gm, "'")
    //     .replace(/b'/gm, '')
    // doc.about = a
    // doc.cgst = 2.5
    // doc.sgst = 2.5
    // doc.discount = 20
    // doc.stock = 100
    // doc.mrp = 0
    // doc.name = doc.medicine_name.replace(/'/gm, '')
    // const id = doc.medicine_name.replace(/ /gm, '-').replace(/\'/gm, '-').replace(/\./gm, '-').toLowerCase()
    // doc.id = id
    // doc.manufacturer_name = doc.manufacturer.name
    // doc.updated_on = new Date()
    // doc.category_id = doc.category.replace(/ /gm, '-').replace(/\//gm, '')

    // delete (doc._id)
    // p = await avail_products.find(o => o.id == id)
    // doc.pos_id = p ? parseInt(p.pos_id) : 0
    // //FIND THE PRODUCT CODE , PRICE,,,,,FROM THE PRODUCT
    // console.log(doc)
    // try {
    //     client.db("sakthi_dev").collection("new_product").updateOne({ _id: docid }, { $set: { doc } }).then(res => {
    //         console.log(res)
    //     })
    // }
    // catch (e) {
    //     console.log(e, idx)
    // }
}

main().catch(console.error);

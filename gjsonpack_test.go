package gjsonpack

import (
	`encoding/json`
	`fmt`
	`testing`
)

var basicJSON = `  {"age":100, "name":{"here":"B\\\"R"},
	"noop":{"what is a wren?":"a bird"},
	"happy":true,"immortal":false,
	"items":[1,2,3,{"tags":[1,2,3],"points":[[1,2],[3,4]]},4,5,6,7],
	"arr":["1",2,"3",{"hello":"world"},"4",5],
	"vals":[1,2,3,{"sadf":"asdf"}],"name":{"first":"tom","last":null},
	"created":"2014-05-16T08:28:06.989Z",
	"loggy":{
		"programmers": [
    	    {
    	        "firstName": "Brett",
    	        "lastName": "McLaughlin",
    	        "email": "aaaa",
				"tag": "good"
    	    },
    	    {
    	        "firstName": "Jason",
    	        "lastName": "Hunter",
    	        "email": "bbbb",
				"tag": "bad"
    	    },
    	    {
    	        "firstName": "Elliotte",
    	        "lastName": "Harold",
    	        "email": "cccc",
				"tag": "good"
    	    },
			{
				"firstName": 1002.3,
				"age": 101
			}
    	]
	},
	"lastly":{"end...ing":"soon","yay":"final"}
}`

// JSON Compress to string
func TestCompress(t *testing.T) {
	jsonMap := make(map[string]interface{}, 0)
	if err := json.Unmarshal([]byte(basicJSON), &jsonMap); err != nil {
		t.Fatal(err)
	}
	packStr, packErr := Pack(jsonMap)
	if packErr != nil {
		t.Fatal(packErr)
	}
	fmt.Println("packStr:", packStr)
}

// Compression string Decompression
func TestDeCompress(t *testing.T) {
	packStr := "noop|what+is+a+wren?|a+bird|happy|items|tags|points|arr|1|3|hello|world|4|loggy|programmers|firstName|Brett|lastName|McLaughlin|email|aaaa|tag|good|Jason|Hunter|bbbb|bad|Elliotte|Harold|cccc|age|name|last|first|tom|immortal|vals|sadf|asdf|created|2014-05-16T08:28:06.989Z|lastly|yay|final|end...ing|soon^1|2|3|1|2|3|1|2|3|4|4|5|6|7|2|5|2T|1|2|3|2S^1002.3^$0|$1|2]|3|-1|4|@1A|1B|1C|$5|@1D|1E|1F]|6|@@1G|1H]|@1I|1J]]]|1K|1L|1M|1N]|7|@8|1O|9|$A|B]|C|1P]|D|$E|@$F|G|H|I|J|K|L|M]|$F|N|H|O|J|P|L|Q]|$F|R|H|S|J|T|L|M]|$F|1V|U|1Q]]]|V|$W|-3|X|Y]|Z|-2|10|@1R|1S|1T|$11|12]]|13|14|15|$16|17|18|19]|U|1U]"
	jsonMap := make(map[string]interface{}, 0)
	unPackErr := Unpack(packStr, &jsonMap)
	if unPackErr != nil {
		return
	}
	fmt.Printf("jsonMap:%#v\n", jsonMap)
}

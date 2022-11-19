package gjsonpack

import (
	"encoding/json"
	"fmt"
	"testing"
)

var basicJSON = `{
    "type": "world",
    "name": "earth",
    "children": [
        {
            "type": "continent",
            "name": "America",
            "children": [
                {
                    "type": "country",
                    "name": "Chile",
                    "children": [
                        {
                            "type": "commune",
                            "name": "Antofagasta"
                        }
                    ]
                }
            ]
        },
        {
            "type": "continent",
            "name": "Europe"
        }
    ]
}`

// JSON Compress to string
func TestCompress(t *testing.T) {
	jsonMap := make(map[string]interface{}, 0)
	if err := json.Unmarshal([]byte(basicJSON), &jsonMap); err != nil {
		t.Fatal(err)
	}
	// pack the big JSON
	packStr, packErr := Pack(jsonMap)
	if packErr != nil {
		t.Fatal(packErr)
	}
	fmt.Println("packStr:", packStr)
}

// Compression string Decompression
func TestDeCompress(t *testing.T) {
	packStr := "type|world|name|earth|children|continent|America|country|Chile|commune|Antofagasta|Europe^^^$0|1|2|3|4|@$0|5|2|6|4|@$0|7|2|8|4|@$0|9|2|A]]]]]|$0|5|2|B]]]"
	jsonMap := make(map[string]interface{}, 0)
	unPackErr := Unpack(packStr, &jsonMap)
	if unPackErr != nil {
		return
	}
	fmt.Printf("jsonMap:%v\n", jsonMap)
}

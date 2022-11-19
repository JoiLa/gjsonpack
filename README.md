# Description

A compression algorithm for JSON

gjsonpack is a GoLang program to pack and unpack JSON data.

It can compress to 55% of original size if the data has a recursive structure. 

take JavaScript transformation to GoLang, source repositories please [see](https://github.com/rgcl/jsonpack).



# How to use?

```bash
go get github.com/JoiLa/gjsonpack
```



# How to compress json

```go
// big JSON
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

jsonMap := make(map[string]interface{}, 0)
if err := json.Unmarshal([]byte(basicJSON), &jsonMap); err != nil {
    t.Fatal(err)
}

// pack the big JSON 
packStr, packErr := Pack(jsonMap)
if packErr != nil {
    fmt.Fprintln(packErr)
}
fmt.Println("packStr:", packStr)
// packStr: type|world|name|earth|children|continent|America|country|Chile|commune|Antofagasta|Europe^^^$0|1|2|3|4|@$0|5|2|6|4|@$0|7|2|8|4|@$0|9|2|A]]]]]|$0|5|2|B]]]

// do something with the packed JSON

```



# How to decompress json

```go
packStr:="type|world|name|earth|children|continent|America|country|Chile|commune|Antofagasta|Europe^^^$0|1|2|3|4|@$0|5|2|6|4|@$0|7|2|8|4|@$0|9|2|A]]]]]|$0|5|2|B]]]"
jsonMap := make(map[string]interface{}, 0)
unPackErr := Unpack(packStr, &jsonMap)
if unPackErr != nil {
    return
}
fmt.Printf("jsonMap:%v\n", jsonMap)
// jsonMap:map[children:[map[children:[map[children:[map[name:Antofagasta type:commune]] name:Chile type:country]] name:America type:continent] map[name:Europe type:continent]] name:earth type:world]

// do something with the unPacked JSON
```


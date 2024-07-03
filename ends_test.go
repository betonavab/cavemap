package cavemap

import (
	"testing"
	//"fmt"
)

func Test_AddLocalSurvey(t *testing.T) {
	var chicoFree = []Station{
        {Id: 159, Name: "START", FromId: -1, Section: "FREEDIVE", Type: START, Depth: -5.4, Lon: -87.447680, Lat: 20.317899, Comment: "START"},
        {Id: 160, Name: "CsFree1", FromId: 159, Section: "FREEDIVE", Type: REAL, Len: 14.5, Azi: 170, Depth: 0, Comment: "near jetty, silty floor"},
        {Id: 162, Name: "CsFree3", FromId: 161, Section: "FREEDIVE", Type: REAL, Len: 10.07, Azi: 197, Depth: 5.7, Comment: "silt, R, zero vis"},
        {Id: 161, Name: "CsFree2", FromId: 160, Section: "FREEDIVE", Type: REAL, Len: 8.2, Azi: 182, Depth: 2.9, Comment: "silt, ceramic"},
        {Id: 164, Name: "CsFree5", FromId: 163, Section: "FREEDIVE", Type: REAL, Len: 2.15, Azi: 201, Depth: 9.4, Comment: "R end"},
        {Id: 163, Name: "CsFree4", FromId: 162, Section: "FREEDIVE", Type: REAL, Len: 5.92, Azi: 177, Depth: 8.4, Comment: ""},
        {Id: 166, Name: "CsFree7", FromId: 165, Section: "FREEDIVE", Type: REAL, Len: 9.02, Azi: 253, Depth: 11.4, Comment: "continues"},
        {Id: 165, Name: "CsFree6", FromId: 164, Section: "FREEDIVE", Type: REAL, Len: 2.95, Azi: 169, Depth: 11.2, Comment: "!E!>Beto2023"},
	}
	var geojson = `{"type":"FeatureCollection","features":[{"type":"Feature","geometry":{"type":"Point","coordinates":[-87.44768,20.317899]},"properties":{"comment":"START","depth":-5.4,"name":"START"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-87.44765585363852,20.317770579459303]},"properties":{"comment":"near jetty, silty floor","depth":0,"name":"CsFree1"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-87.44765859803134,20.317696880010676]},"properties":{"comment":"silt, ceramic","depth":2.9,"name":"CsFree2"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-87.44768683238345,20.317610275437655]},"properties":{"comment":"silt, R, zero vis","depth":5.7,"name":"CsFree3"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-87.44768386116395,20.317557108561907]},"properties":{"comment":"","depth":8.4,"name":"CsFree4"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-87.447691250075,20.31753905739721]},"properties":{"comment":"R end","depth":9.4,"name":"CsFree5"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-87.44768585206599,20.31751301484034]},"properties":{"comment":"!E!\u003eBeto2023","depth":11.2,"name":"CsFree6"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-87.44776857301903,20.31748929797647]},"properties":{"comment":"continues","depth":11.4,"name":"CsFree7"}},{"type":"Feature","geometry":{"type":"GeometryCollection","geometries":[{"type":"LineString","coordinates":[[-87.44768,20.317899],[-87.44765585363852,20.317770579459303]]},{"type":"LineString","coordinates":[[-87.44765585363852,20.317770579459303],[-87.44765859803134,20.317696880010676]]},{"type":"LineString","coordinates":[[-87.44765859803134,20.317696880010676],[-87.44768683238345,20.317610275437655]]},{"type":"LineString","coordinates":[[-87.44768683238345,20.317610275437655],[-87.44768386116395,20.317557108561907]]},{"type":"LineString","coordinates":[[-87.44768386116395,20.317557108561907],[-87.447691250075,20.31753905739721]]},{"type":"LineString","coordinates":[[-87.447691250075,20.31753905739721],[-87.44768585206599,20.31751301484034]]},{"type":"LineString","coordinates":[[-87.44768585206599,20.31751301484034],[-87.44776857301903,20.31748929797647]]}]},"properties":{"name":"START"}}]}`
        m := New("Chico")

	if err := m.AddLocalSurvey(chicoFree); err !=nil {
		t.Error("cant add chicoFree")
	}
	if len(m.DB) != len(chicoFree) {
		t.Errorf("wrong number of stations added %v, want %v",len(m.DB),len(chicoFree))
	}
	
	for _,s:=range chicoFree {
		id,ok:=m.getStationId(s.Name)
		if !ok {
			t.Errorf("cant find station %v",s.Name)
		} else {
			if id != s.Id {
				t.Errorf("station got wrong id %v,want %v",id,s.Id)
			}
		}
	}
	m.PropagateLocation()
	out,err:=m.Marshal()
	if err != nil {
		t.Errorf("failed to Marshall %v",m)
	}
	if len(geojson) != len(out) {
		t.Errorf("got different len GEOJSON %v,want %v",len(out),len(geojson))
	}
}

func Test_AddSurvey(t *testing.T) {
	
	var text = `
#Survey from the tree in front of the entrance, to the start of the
#cave line.
#
#Ariane: "GPS to water" light red color
#
#TODO: segment 1 is an estimate, survey propertly
#
#Name	Azi	Len	Depth	Commets
auto
0	-87.451223	20.317874
1	108	18	0.0	roof inside cavern
2	79	4.55	2.63	ceiling, calcite, R
3	122	2.16	4.06	ceiling small dome
`
	var geojson = `{"type":"FeatureCollection","features":[{"type":"Feature","geometry":{"type":"Point","coordinates":[-87.451223,20.317874]},"properties":{"comment":"","depth":0,"name":"GPS0"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-87.4510588305111,20.31782397690468]},"properties":{"comment":"roof inside cavern","depth":0,"name":"GPS1"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-87.45101599818992,20.317831784638138]},"properties":{"comment":"ceiling, calcite, R","depth":2.63,"name":"GPS2"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-87.45099843158461,20.31782149077183]},"properties":{"comment":"ceiling small dome","depth":4.06,"name":"GPS3"}},{"type":"Feature","geometry":{"type":"GeometryCollection","geometries":[{"type":"LineString","coordinates":[[-87.451223,20.317874],[-87.4510588305111,20.31782397690468]]},{"type":"LineString","coordinates":[[-87.4510588305111,20.31782397690468],[-87.45101599818992,20.317831784638138]]},{"type":"LineString","coordinates":[[-87.45101599818992,20.317831784638138],[-87.45099843158461,20.31782149077183]]}]},"properties":{"name":"GPS0"}}]}a`
        m := New("Test")
	srv,start,err := m.ParseSurvey([]byte(text),"GPS")
	if err != nil {
		t.Errorf("failed to parse bytes")
	}
	if srv == nil {
		t.Errorf("nil survey")
	}
	if start != "START" {
		t.Errorf("bad start type %v, want %v",start,"START")
	}
	if len(srv) != 4 {
		t.Errorf("different number of station %v, want %v",len(srv),4)
	}
	if err:=m.ValidSurvey(srv); err!=nil {
		t.Errorf("invalid survey")
	}
	if err:=m.AddSurvey(srv,start); err!=nil {
		t.Errorf("failed to add survey %v",err)
	}
	if len(m.DB) != 4 {
		t.Errorf("different number of station %v, want %v",len(m.DB),4)
	}

	m.PropagateLocation()
	out,err:=m.Marshal()
	if err != nil {
		t.Errorf("failed to Marshall %v",m)
	}
	if len(geojson) < len(out) {
		t.Errorf("got different len GEOJSON %v,want %v",len(out),len(geojson))
	}
}

package cavemap
//Package cavemap provides a very simple implementation of underwater cave survey.
//A map is used to hold a series of surveys, a survey is a sequence of stations,
//and stations are separated from each other by a distance and an azimuth.
//Both maps and surveys are stored in files. Maps are formmated as JSON files, and surveys
//use a simple syntax. 
import (
	"fmt"
	"math"
	"io"
	"sort"
	"regexp"
	"slices"
	"sync"
	"strconv"
	"strings"

	geojson "github.com/paulmach/go.geojson"
)

//A START station is positioned by its coordinate, while a REAL
//station is located by following a survey from a START station
const (
	START = iota
	REAL
)

//A Station represent a position inside a map. You get to it by
//traversing a survey file that constains it.
type Station struct {
	Id      int
	Name    string
	FromId  int
	Section string
	Type    int
	Len     float64
	Azi     float64
	Depth   float64
	Lon     float64
	Lat     float64
	Comment string
}

func (s *Station) String() string {
	if s.Type == START {
		return fmt.Sprintf("%.8v/%.8v", s.Lon, s.Lat)
	}
	return fmt.Sprintf("%s", s.Name)
}

//A Map represent a series of surveys. A map can be loaded from a file, as well as
//stored into one. Surveys are added to the map one at a time. 
type Map struct {
	Name string
	mu   sync.Mutex
	DB   map[int]*Station
}

func (m *Map) String() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return fmt.Sprintf("%s %d stations", m.Name, len(m.DB))
}

func New(name string) *Map {
	m := &Map{}
	m.Name = name
	m.DB = make(map[int]*Station)
	return m
}
 
//AddLocalSurvey adds a survey[] to a map. Survey is often times a []Station{..} that is
//declare globally. Minimal validation are done on the survey
func (m *Map) AddLocalSurvey(survey []Station) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, s := range survey {
		if _, ok := m.DB[s.Id]; ok {
			return fmt.Errorf("station %s already in DB with id %d",s.Name, s.Id)
		}
		m.DB[s.Id] = &survey[i]
	}
	return nil
}

func  averageAzimuth(x,y float64) float64 {
        if x < y {
                x,y = y,x
        }
        if int(x)/90 == 3 && int(y)/90 == 0 {
                a:=x + (360-x+y)/2
                if a >= 360 {
                        return a - 360
                }
                return a
        }
        return x - (x-y)/2
} 

func reverseAzimuth(x float64) float64 {
	if x < 180 {
		return x+180
	}
	return x-180
}

//ParseSurvey reads a survey from a text. It uses a very simple syntax
//of \t separates fields. Comments start with #, and the first line is the
//station the survey starts from. Other lines describe how to arrive to each
//new stations inside the survey. A new station position is described by one or two
//azimuths and a distance. A station has a depth, and also a comment field. 
//ParseSurvey does not add the survey to the map. For that use AddSurvey
func (m *Map) ParseSurvey(text []byte,prefix string) ([]Station, string, error) {
	var srv []Station
	start := "START"
	label:=""
	reverse:=false
	for n, line := range strings.Split(string(text), "\n") {
		if line == "" {
			continue
		}
		if string(line[0]) == "#" {
			continue
		}
		field 	:= strings.Split(line, "\t")
		report := func (msg string, err error) ([]Station, string, error) {
			if debug {
				for i,f := range field {
					fmt.Printf("field[%v]=%v\n",i,f)
				}
			}
			return srv,start,fmt.Errorf("line %v parsing %s: %s",n+1,msg,err) 
		}

		// Handle trailing \t at the end of comment
		// and no comment
		nfield:=len(field)
		hasempty:=false
		hascomment:=false
		last:=-1
		for i,f:=range field {
			if f == "" {
				hasempty=true
				continue
			}
			last=i
			if i!=0 && f!="-" {
				_, err := strconv.ParseFloat(field[last], 64)
				if err != nil {
					hascomment=true
				}
			}
		}
		
		if hasempty && last != -1 {
			_, err := strconv.ParseFloat(field[last], 64)
			if err == nil {
				if debug {
					fmt.Println("compacting multiple empty fields into one")
				}
				last++
				field[last]=""
			}
			nfield=last+1
		} else if !hascomment {
			if nfield >= 4 {
				if debug {
					fmt.Println("adding empty comment")
				}
				field=append(field,"")
				nfield++
			}
		}
		if debug {
		  for i,f := range field {
			if i < nfield {
				if debug {
					fmt.Printf("field[%v]=%v\n",i,f)
				}
			}
		  }
		  fmt.Printf("--- hascomment %v hasempty %v\n",hascomment, hasempty)
		}

		switch nfield {
		case 1:
			if field[0] == "auto" && prefix != "" {
				label=prefix
				continue
			}
			if field[0] == "reverse" {
				reverse=true
				continue
			}
			start = field[0]
		case 3:
			name := label+field[0]
			lon, err := strconv.ParseFloat(field[1], 64)
			if err != nil {
				return report("longitute",err)
			}
			lat, err := strconv.ParseFloat(field[2], 64)
			if err != nil {
				return report("latitude",err)
			}
			srv = append(srv, Station{Name: name, Type: START, Lon: lon, Lat: lat})
		case 5:
			if reverse {
				field[1],field[2] = field[2],field[1]
				field[2],field[3] = field[3],field[2]
			}
			name := label+field[0]
			azi, err := strconv.ParseFloat(field[1], 64)
			if err != nil {
				return report("azimuth",err)
			}
			len, err := strconv.ParseFloat(field[2], 64)
			if err != nil {
				return report("length",err)
			}
			depth, err := strconv.ParseFloat(field[3], 64)
			if err != nil {
				return report("depth",err)
			}
			comment := field[4]
			srv = append(srv, Station{Name: name, Type: REAL,
				Azi: azi, Len: len, Depth: depth, Comment: comment})
		case 6:
			if reverse {
				field[1],field[2] = field[2],field[1]
				field[2],field[3] = field[3],field[2]
				field[3],field[4] = field[4],field[3]
			}
			name := label+field[0]
			azi1, err := strconv.ParseFloat(field[1], 64)
			if err != nil {
				return report("azimuth1",err)
			}
			len, err := strconv.ParseFloat(field[2], 64)
			if err != nil {
				return report("length",err)
			}
			azi := azi1
			if field[3] != "-" {
				azi2, err := strconv.ParseFloat(field[3], 64)
				if err != nil {
					return report("azimuth2",err)
				}
				azi = averageAzimuth(azi1,azi2)
			}
			depth, err := strconv.ParseFloat(field[4], 64)
			if err != nil {
				return report("depth",err)
			}
			comment := field[5]
			srv = append(srv, Station{Name: name, Type: REAL,
				Azi: azi, Len: len, Depth: depth, Comment: comment})
		default:
			return report("line",fmt.Errorf("wrong number %v of fields %v",len(field),field))
		}
	}
	if reverse {
		slices.Reverse(srv)
		if prefix != ""  {
			for i,_ := range srv {
				srv[i].Name=fmt.Sprintf("%s%v",prefix,i+1)
				srv[i].Azi = reverseAzimuth(srv[i].Azi)
			}
		}
	}
	return srv, start, nil
}

func (m *Map) ValidSurvey(srv []Station) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range srv {
		for _, s1 := range m.DB {
			if s.Name != "START" && s.Name == s1.Name {
				return fmt.Errorf("duplicate name %s",s.Name)
			}
		}
	}
	return nil
}

func (m *Map) PrintSurvey(start string, srv []Station) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if start != "START" {
		fmt.Println(start)
	}
	for _, s := range srv {
		if s.Type == START {
			fmt.Printf("%v\t%.8v\t%.8v\n",s.Name, s.Lon, s.Lat)
		} else if s.Type == REAL {
			fmt.Printf("%v\t%v\t%v\t%v\t%v\n", s.Name, s.Azi, s.Len, s.Depth, s.Comment)
		}
	}
}
//PrintSurveyAsSRV prints the survey in Walls format
func (m *Map) PrintSurveyAsSRV(start string, srv []Station) error {
	var from string
	var fromDepth float64
	printHeader:= func () {
		fmt.Printf("#UNITS Meters ORDER=DA TAPE=SS\n")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if start != "START" {
		if len(srv) == 0 {
			return nil	
		}
		s:=srv[0]
		from=start
		fromId,ok:=m.getStationId(from)
		if !ok {
			return fmt.Errorf("unknown from station %v",from)
		}
		if s, ok := m.DB[fromId]; ok {
			fromDepth=s.Depth
		} 

		printHeader()
		fmt.Printf("%v\t%v\t%v\t%v\t%v\t%v\t;%v\n",
		  from,s.Name,s.Len,s.Azi,fromDepth,s.Depth,s.Comment)
	} else {
		printHeader()
	}

	for i, s := range srv {
		if i > 0 {
			fmt.Printf("%v\t%v\t%v\t%v\t%v\t%v\t;%v\n",
			  from,s.Name,s.Len,s.Azi,fromDepth,s.Depth,s.Comment)
		}
		from=s.Name
		fromDepth=s.Depth
	}
	return nil
}

//Caller should have m.mu locked
func (m *Map) getStationId(name string) (int, bool) {
	for _, s := range m.DB {
		if s.Name == name {
			return s.Id, true
		}
	}
	return -1, false
}

//AddSurvey commits a survey to the map. It's important to parse
//and validate the survey before.
func (m *Map) AddSurvey(srv []Station, start string) error {
	if srv == nil || len(srv) <= 0 {
		return fmt.Errorf("can't add empty survey")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	from := -1
	if start != "START" {
		var ok bool
		from, ok = m.getStationId(start)
		if !ok || from == -1 {
			return fmt.Errorf("unknow station %v",start)
		}
	}
	if debug {
		fmt.Printf("from station %v %v\n",start,from)
	}

	maxid := 0
	for _, s := range m.DB {
		if s.Id > maxid {
			maxid = s.Id
		}
	}
	if debug {
		fmt.Printf("maxid %v\n", maxid)
	}
	for i, _ := range srv {
		maxid++
		srv[i].Id = maxid
		srv[i].FromId = from
		from = maxid
	}

	for i, s := range srv {
		if _, ok := m.DB[s.Id]; ok {
			return fmt.Errorf("%v already in DB",m.DB[s.Id])
		}
		if debug {
			fmt.Printf("adding[%d] %s\n",s.Id,&s)
		}
		m.DB[s.Id] = &srv[i]
	}
	return nil
}

//Helper function to recursively travel the map. Pre is called before
//visiting the stations that can be reach from current station (pre order)
func (m *Map) forEachStation(Id int,pre func (from, s *Station)) {
	//TODO: Should currency be on this function or on the callers
	// as in ShowGo 
	for _, s := range m.DB {
		if s.FromId == Id {
			if pre != nil {
				pre(m.DB[s.FromId],s)
			}
			m.forEachStation(s.Id,pre)
		}
	}
}

//Show prints the map to stdout 
func (m *Map) Show() {
	var wg sync.WaitGroup
	fmt.Printf("Map: %s\n", m)
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.DB {
		if s.Type == START {
			wg.Add(1)
			go func (s *Station) {
				defer wg.Done()
				fmt.Printf("%s: \n", s)
				m.forEachStation(s.Id,func (f,s *Station){fmt.Printf("%s->%s\n",f ,s)})
			} (s)
		}
	}
	wg.Wait()
}
//ShowGo prints the map to stdout running currently
func (m *Map) ShowGo() {
	complete:=make (chan chan string)
	var wg sync.WaitGroup
	fmt.Printf("Map: %s\n", m)
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.DB {
		if s.Type == START {
			wg.Add(1)
			out:=make(chan string,len(m.DB))
			go func (s *Station, out chan string) {
				defer wg.Done()
				out <-fmt.Sprintf("%s: \n", s)
				m.forEachStation(s.Id,func (f,s *Station){out<-fmt.Sprintf("%s->%s\n",f ,s)})
				complete <- out
				close(out)
			} (s,out)
		}
	}
	go func () {
		wg.Wait()
		close (complete)
	}()

	for out := range complete {
		for m := range out {
			fmt.Printf(m)
		}
	}
}



//http://www.movable-type.co.uk/scripts/latlon.html
//
//const φ2 = Math.asin( Math.sin(φ1)*Math.cos(d/R) +
//                      Math.cos(φ1)*Math.sin(d/R)*Math.cos(brng) );
//const λ2 = λ1 + Math.atan2(Math.sin(brng)*Math.sin(d/R)*Math.cos(φ1),
//                      Math.cos(d/R)-Math.sin(φ1)*Math.sin(φ2))
//TODO: what about delta btw the depths

func advLonLat(lon, lat, azi, len float64) (float64, float64) {
	const R = 6371e3

	radlat1 := float64(math.Pi*lat/180)
	radlon1:= float64(math.Pi*lon/180)
	radazi 	:=  float64(math.Pi*azi/180)

	radlat2 := math.Asin( math.Sin(radlat1)*math.Cos(len/R)+
			math.Cos(radlat1)*math.Sin(len/R)*math.Cos(radazi))
	radlon2 := radlon1 + math.Atan2(math.Sin(radazi)*math.Sin(len/R)*math.Cos(radlat1),
				math.Cos(len/R)-math.Sin(radlat1)*math.Sin(radlat2))
			
	return radlon2 * 180 /math.Pi, radlat2 * 180 / math.Pi
}

//PropagateLocation computes stations location based on the map
//START stations and the survey data
func (m *Map) PropagateLocation() {
	m.mu.Lock()
	defer m.mu.Unlock()
        for _, s := range m.DB {
                if s.Type == START {
			updateStation:=func (f,s *Station) {
			 if s.Lon == 0 && s.Lat == 0 {
				s.Lon, s.Lat = advLonLat(f.Lon,f.Lat,s.Azi,s.Len)
				if debug {
					fmt.Printf("update[%v] %.8v/%.8v\n",s.Name,s.Lon,s.Lat)
				}
			 }
			}
                        m.forEachStation(s.Id,updateStation)
                }
        }
}

type byName []string
func (s byName) Len() int 	{return len(s)}
func (s byName) Less(i,j int) (ret bool) {
	ret = s[i] < s[j]
	if s[i] == "START" {
		return true
	}
	if s[j] == "START" {
		return false
	}
	re,err := regexp.Compile("[a-zA-z]+[0-9]+")
	if err!= nil {
		return
	}
	if !re.MatchString(s[i]) || 
		!re.MatchString(s[j]) {
		return
	}
	re,err = regexp.Compile("[a-zA-z]+")
	if err!= nil {
		return
	}
	p1:=re.Find([]byte(s[i]))
	p2:=re.Find([]byte(s[j]))
	if p1 == nil || p2 == nil{
		return
	}
	if string(p1) != string(p2) {
		return 
	}
	re,err = regexp.Compile("[0-9]+")
	if err!= nil {
		return
	}
	n1:=re.Find([]byte(s[i]))
	n2:=re.Find([]byte(s[j]))
	if n1 == nil || n2 == nil{
		return
	}
	f1,err := strconv.ParseFloat(string(n1),64)
	if err!= nil {
		return
	}
	f2,err := strconv.ParseFloat(string(n2),64)
	if err!= nil {
		return
	}
	return f1 < f2
}
func (s byName) Swap(i,j int) 	{ s[i],s[j]=s[j],s[i]}

//Marshal returns a string which contains a GEOJSON representaton of 
//the map.
func (m *Map) Marshal() (string, error ){
	//TODO: Produce a more palate version of the map
	m.mu.Lock()
	defer m.mu.Unlock()
	fc := geojson.NewFeatureCollection()
	var name[]string
	nameToStation := make (map[string]*Station)
	for i,s := range m.DB {
		name = append(name,s.Name)
		nameToStation[s.Name]=m.DB[i]
	}
	sort.Sort(byName(name))
	for _,n := range name {
		s:=nameToStation[n]
		f:=geojson.NewPointFeature([]float64{s.Lon, s.Lat})
		f.Properties["name"]=n
		f.Properties["depth"]=s.Depth
		f.Properties["comment"]=s.Comment
		fc.AddFeature(f)
	}
	var reach []int
	for _, s := range m.DB {
		if s.Type == START {
			var co [][]float64
			reach=append(reach,s.Id)
			if debug {
				fmt.Printf("%s\n", s)
			}
			appendStation := func (f,s *Station) {
			 if debug {
				fmt.Printf("%s->%s\n", f,s)
			 }
			 co = append(co,[]float64{f.Lon,f.Lat})
			 co = append(co,[]float64{s.Lon,s.Lat})
			 reach = append(reach,s.Id)
			}
			m.forEachStation(s.Id,appendStation)

			var lines []*geojson.Geometry
			for i:=0; i < len(co) ; i+=2 {
				if debug {
					fmt.Println(co[i:i+2])
				}
				//fc.AddFeature(geojson.NewLineStringFeature(co[i:i+2]))
				lg:=geojson.NewLineStringGeometry (co[i:i+2])
				lines = append(lines,lg)
			}
			gc:=geojson.NewCollectionGeometry(lines...)
			lf:=geojson.NewFeature(gc)
			lf.Properties["name"]=s.Name
			fc.AddFeature(lf)
		}
	}
	//sanity check
	if len(reach) != len(m.DB) {
		var miss []int
		for _,s := range m.DB {
			found:=false
			for _,Id:=range reach {
				if Id == s.Id {
					found=true
					break
				}
			}
			if !found {
				miss=append(miss,s.Id)
			}
		}
		for _,Id := range miss {
			s:=m.DB[Id]
			fmt.Printf("missed %v[%v] fromID %v\n",Id,s.Name,s.FromId)
		}
	}

	rawJSON, err := fc.MarshalJSON()
	if err != nil {
		return "",fmt.Errorf("failed to marshal: %v",err)
	}

	return string(rawJSON),nil
}
var debug bool
var dwriter io.Writer

// EnableDebug turn debugging on
func EnableDebug(w io.Writer) {
        debug = true
        dwriter = w
}
        
// DisableDebug turn debugging off
func DisableDebug() {
        debug = false
} 

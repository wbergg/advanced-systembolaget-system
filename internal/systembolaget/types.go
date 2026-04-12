package systembolaget

type Config struct {
	APIKey string `json:"api_key"`
}

type ProductImage struct {
	ImageURL string `json:"imageUrl"`
	FileType string `json:"fileType"`
}

type Product struct {
	ProductID       string  `json:"productId"`
	ProductNumber   string  `json:"productNumber"`
	ProductNameBold string  `json:"productNameBold"`
	ProductNameThin *string `json:"productNameThin"`
	ProducerName    string  `json:"producerName"`
	Price           float64 `json:"price"`
	Volume          float64 `json:"volume"`
	VolumeText      string  `json:"volumeText"`
	AlcoholPercent  float64 `json:"alcoholPercentage"`
	Country         string  `json:"country"`
	CategoryLevel1  string  `json:"categoryLevel1"`
	CategoryLevel2  string  `json:"categoryLevel2"`
	AssortmentText  string  `json:"assortmentText"`
	Taste           string  `json:"taste"`
	Usage           string  `json:"usage"`
	IsOrganic              bool    `json:"isOrganic"`
	IsNews                 bool    `json:"isNews"`
	IsDiscontinued         bool    `json:"isDiscontinued"`
	IsCompletelyOutOfStock bool    `json:"isCompletelyOutOfStock"`
	IsTemporaryOutOfStock  bool    `json:"isTemporaryOutOfStock"`
	PackagingLevel1        string  `json:"packagingLevel1"`
	Assortment             string  `json:"assortment"`
	ProductLaunchDate      string  `json:"productLaunchDate"`
	IsRegionalRestricted      bool    `json:"isRegionalRestricted"`
	RestrictedParcelQuantity int     `json:"restrictedParcelQuantity"`
	Vintage                  *string `json:"vintage"`
	ImageURL                 string  `json:"imageUrl,omitempty"`
}

type RawProduct struct {
	Product
	Images []ProductImage `json:"images"`
}

type SearchResponse struct {
	Metadata struct {
		DocCount   int `json:"docCount"`
		NextPage   int `json:"nextPage"`
		TotalPages int `json:"totalPages"`
	} `json:"metadata"`
	Products []RawProduct `json:"products"`
}

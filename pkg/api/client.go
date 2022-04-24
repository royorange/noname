package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type API struct {
	Cookie string
	client *http.Client
	signer *Signer

	ddmcUid string

	address   *Address
	debugTime string
}

func NewAPI(cookie string) (*API, error) {
	if len(cookie) == 0 {
		return nil, errors.New("无效的cookie")
	}

	signer, err := NewSigner("./sign.js")
	if err != nil {
		return nil, err
	}

	return &API{
		Cookie: cookie,
		client: http.DefaultClient,
		signer: signer,
	}, nil
}

func (api *API) SetAddress(address Address) *API {
	api.address = &address
	return api
}

func (api *API) SetDebugTime(time string) *API {
	api.debugTime = time
	return api
}

func (api *API) getTime() string {
	if len(api.debugTime) > 0 {
		return api.debugTime
	}

	return strconv.FormatInt(time.Now().Unix(), 10)
}

func (api *API) getLocation() ([]string, error) {
	if api.address == nil {
		return nil, errors.New("请先使用SetAddress设置地址")
	}

	return []string{
		fmt.Sprint(api.address.Location.Location[0]),
		fmt.Sprint(api.address.Location.Location[1]),
	}, nil
}

func (api *API) UserDetail() (*UserDetail, error) {
	url, err := url.ParseRequestURI("https://sunquan.api.ddxq.mobi/api/v1/user/detail/")
	if err != nil {
		return nil, err
	}

	var query = url.Query()
	query.Set("api_version", "9.50.0")
	query.Set("app_version", "2.83.0")
	query.Set("applet_source", "")
	query.Set("channel", "applet")
	query.Set("app_client_id", "4")
	url.RawQuery = query.Encode()

	request, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}

	var header = api.newBaseHeader()
	if err != nil {
		return nil, err
	}

	header.Set("host", "sunquan.api.ddxq.mobi")
	request.Header = header

	var detail = new(UserDetail)
	err = api.do(request, nil, detail)
	if err != nil {
		return nil, err
	}

	api.ddmcUid = detail.UserInfo.ID

	return detail, nil
}

func (api *API) UserAddress() (*UserAddress, error) {
	url, err := url.ParseRequestURI("https://sunquan.api.ddxq.mobi/api/v1/user/address/")
	if err != nil {
		return nil, err
	}

	var query = url.Query()
	query.Set("api_version", "9.50.0")
	query.Set("app_version", "2.83.0")
	query.Set("applet_source", "")
	query.Set("channel", "applet")
	query.Set("app_client_id", "4")
	url.RawQuery = query.Encode()

	request, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}
	header, err := api.newHeader()
	if err != nil {
		return nil, err
	}

	header.Set("host", "sunquan.api.ddxq.mobi")
	request.Header = header
	var address = new(UserAddress)
	err = api.do(request, nil, address)
	if err != nil {
		return nil, err
	}

	return address, nil
}

func (api *API) Cart() (*CartInfo, error) {
	if api.address == nil {
		return nil, errors.New("需先使用SetAddress绑定地址信息")
	}

	url, err := url.ParseRequestURI("https://maicai.api.ddxq.mobi/cart/index")
	if err != nil {
		return nil, err
	}

	var query = url.Query()
	query.Set("station_id", api.address.StationInfo.ID)
	query.Set("is_load", "1")
	query.Set("api_version", "9.50.0")
	query.Set("app_version", "2.83.0")
	query.Set("applet_source", "")
	query.Set("channel", "applet")
	query.Set("app_client_id", "4")
	url.RawQuery = query.Encode()

	request, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}

	header, err := api.newHeader()
	if err != nil {
		return nil, err
	}
	request.Header = header
	var cart = new(CartInfo)
	err = api.do(request, nil, cart)
	if err != nil {
		return nil, err
	}

	return cart, nil
}

func (api *API) GetMultiReverseTime(products []ProductListItem) (*MultiReserveTime, error) {
	if api.address == nil {
		return nil, errors.New("需先使用SetAddress绑定地址信息")
	}

	data, err := json.Marshal([]interface{}{products})
	if err != nil {
		return nil, err
	}

	params := api.newURLEncodedForm()
	params.Set("station_id", api.address.StationInfo.ID)
	params.Set("address_id", api.address.ID)
	params.Set("group_config_id", ``)
	params.Set("products", string(data))
	params.Set("isBridge", `false`)

	url, err := url.ParseRequestURI("https://maicai.api.ddxq.mobi/order/getMultiReserveTime")
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest(http.MethodPost, url.String(), nil)
	if err != nil {
		return nil, err
	}

	header, err := api.newHeader()
	if err != nil {
		return nil, err
	}

	request.Header = header
	var times = new(MultiReserveTime)
	err = api.do(request, params, times)
	if err != nil {
		return nil, err
	}

	return times, nil
}

func (api *API) UpdateCheck(productId string, cartId string) (*CartInfo, error) {

	var data = struct {
		ID      string     `json:"id"`
		CartID  string     `json:"cart_id"`
		IsCheck bool       `json:"is_check"`
		Sizes   []struct{} `json:"sizes"`
	}{
		ID:      productId,
		CartID:  cartId,
		IsCheck: true,
		Sizes:   []struct{}{},
	}

	packagesData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var urlForm = api.newURLEncodedForm()
	urlForm.Set("product", string(packagesData))
	urlForm.Set("is_load", "1")
	urlForm.Set("ab_config", `{"key_onion":"D","key_cart_discount_price":"C"}`)

	url, err := url.ParseRequestURI("https://maicai.api.ddxq.mobi/cart/updateCheck")
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest(http.MethodPost, url.String(), nil)
	if err != nil {
		return nil, err
	}

	header, err := api.newHeader()
	if err != nil {
		return nil, err
	}

	request.Header = header
	var cart = new(CartInfo)
	err = api.do(request, urlForm, cart)
	if err != nil {
		return nil, err
	}

	return cart, nil
}

func (api *API) CheckOrder(productList ProductList) (*CheckOrder, error) {
	if len(productList.Products) == 0 {
		return nil, errors.New("没有可购买商品")
	}

	for i := range productList.Products {
		product := &productList.Products[i]
		product.TotalOriginMoney = product.TotalOriginPrice
		product.TotalMoney = product.TotalPrice
	}

	type ReservedTime struct {
		ReservedTimeStart *int64 `json:"reserved_time_start"`
		ReservedTimeEnd   *int64 `json:"reserved_time_end"`
	}
	var data = struct {
		ProductList
		ReservedTime ReservedTime `json:"reserved_time"`
	}{
		ProductList:  productList,
		ReservedTime: ReservedTime{},
	}

	packagesData, err := json.Marshal([]interface{}{data})
	if err != nil {
		return nil, err
	}

	var urlForm = api.newURLEncodedForm()
	urlForm.Set("user_ticket_id", "default")
	urlForm.Set("freight_ticket_id", "default")
	urlForm.Set("is_use_point", "0")
	urlForm.Set("is_use_balance", "0")
	urlForm.Set("is_buy_vip", "0")
	urlForm.Set("coupons_id", "")
	urlForm.Set("is_buy_coupons", "0")
	urlForm.Set("packages", string(packagesData))

	url, err := url.ParseRequestURI("https://maicai.api.ddxq.mobi/order/checkOrder")
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest(http.MethodPost, url.String(), nil)
	if err != nil {
		return nil, err
	}

	header, err := api.newHeader()
	if err != nil {
		return nil, err
	}

	request.Header = header
	var checkOrder = new(CheckOrder)
	err = api.do(request, urlForm, checkOrder)
	if err != nil {
		return nil, err
	}

	return checkOrder, nil
}

func (api *API) AddNewOrder(payType int, cartInfo *CartInfo, reserveTime ReserveTime, checkOrder *CheckOrder) (*AddNewOrder, error) {
	if api.address == nil {
		return nil, errors.New("需先使用SetAddress绑定地址信息")
	}

	var payment = struct {
		ReservedTimeStart    int64       `json:"reserved_time_start"`
		ReservedTimeEnd      int64       `json:"reserved_time_end"`
		Price                string      `json:"price"`
		FreightDiscountMoney string      `json:"freight_discount_money"`
		FreightMoney         string      `json:"freight_money"`
		OrderFreight         string      `json:"order_freight"`
		ParentOrderSign      string      `json:"parent_order_sign"`
		ProductType          int         `json:"product_type"`
		AddressID            string      `json:"address_id"`
		FormID               string      `json:"form_id"`
		ReceiptWithoutSku    interface{} `json:"receipt_without_sku"`
		PayType              int         `json:"pay_type"`
		UserTicketID         string      `json:"user_ticket_id"`
		VipMoney             string      `json:"vip_money"`
		VipBuyUserTicketID   string      `json:"vip_buy_user_ticket_id"`
		CouponsMoney         string      `json:"coupons_money"`
		CouponsID            string      `json:"coupons_id"`
	}{
		ReservedTimeStart:    reserveTime.StartTimestamp,
		ReservedTimeEnd:      reserveTime.EndTimestamp,
		ParentOrderSign:      cartInfo.ParentOrderInfo.ParentOrderSign,
		AddressID:            api.address.ID,
		PayType:              payType,
		ProductType:          1,
		FormID:               strings.ReplaceAll(uuid.New().String(), "-", ""),
		ReceiptWithoutSku:    nil,
		VipMoney:             "",
		VipBuyUserTicketID:   "",
		CouponsMoney:         "",
		CouponsID:            "",
		Price:                checkOrder.Order.TotalMoney,
		FreightDiscountMoney: checkOrder.Order.FreightDiscountMoney,
		FreightMoney:         checkOrder.Order.FreightMoney,
		OrderFreight:         checkOrder.Order.Freights[0].Freight.FreightRealMoney,
		UserTicketID:         checkOrder.Order.DefaultCoupon.ID,
	}

	if len(payment.FreightDiscountMoney) == 0 {
		payment.FreightDiscountMoney = "0.00"
	}

	var pkg = struct {
		ProductList
		ReservedTimeStart    int64  `json:"reserved_time_start"`
		ReservedTimeEnd      int64  `json:"reserved_time_end"`
		EtaTraceID           string `json:"eta_trace_id"`
		SoonArrival          string `json:"soon_arrival"`
		FirstSelectedBigTime int64  `json:"first_selected_big_time"`
		ReceiptWithoutSku    int    `json:"receipt_without_sku"`
	}{
		ProductList:          cartInfo.NewOrderProductList[0],
		ReservedTimeStart:    reserveTime.StartTimestamp,
		ReservedTimeEnd:      payment.ReservedTimeEnd,
		EtaTraceID:           "",
		SoonArrival:          "",
		FirstSelectedBigTime: 0,
		ReceiptWithoutSku:    0,
	}

	data, err := json.Marshal(map[string]interface{}{
		"payment_order": payment,
		"packages":      []interface{}{pkg},
	})

	fmt.Println(string(data))

	if err != nil {
		return nil, err
	}

	var params = api.newURLEncodedForm()

	params.Set("showMsg", "false")
	params.Set("showData", "true")
	params.Set("ab_config", `{"key_onion": "C"}`)
	params.Set("package_order", string(data))

	url, err := url.ParseRequestURI("https://maicai.api.ddxq.mobi/order/addNewOrder")
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest(http.MethodPost, url.String(), nil)
	if err != nil {
		return nil, err
	}

	header, err := api.newHeader()
	if err != nil {
		return nil, err
	}

	for k, v := range header {
		fmt.Println(k, "    ", v[0])
	}

	request.Header = header
	var addNewOrder = new(AddNewOrder)
	err = api.do(request, params, addNewOrder, true)
	if err != nil {
		return nil, err
	}

	return addNewOrder, nil
}

func (api *API) newBaseHeader() http.Header {

	header := http.Header{}
	header.Set("host", "maicai.api.ddxq.mobi")
	header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 11_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E217 MicroMessenger/6.8.0(0x16080000) NetType/WIFI Language/en Branch/Br_trunk MiniProgramEnv/Mac")
	header.Set("content-type", "application/x-www-form-urlencoded")
	header.Set("Referer", "https://servicewechat.com/wx1e113254eda17715/425/page-frame.html")

	header.Set("ddmc-api-version", "9.50.0")
	header.Set("ddmc-app-client-id", "4")
	header.Set("ddmc-build-version", "2.83.0")
	header.Set("ddmc-channel", "applet")
	header.Set("ddmc-os-version", "[object Undefined]")

	header.Set("ddmc-ip", "")
	header.Set("ddmc-time", api.getTime())

	header.Set("ddmc-device-id", "osP8I0RgncVIhrJLWwUCb0gi9uDQ")
	header.Set("Cookie", api.Cookie)

	return header
}

func (api *API) newHeader() (http.Header, error) {
	header := api.newBaseHeader()

	if len(api.ddmcUid) == 0 {
		return nil, errors.New("用户id未设置")
	}

	if len(api.ddmcUid) > 0 {
		header.Set("ddmc-uid", api.ddmcUid)
	}

	if api.address != nil {
		header.Set("ddmc-station-id", api.address.StationInfo.ID)
		header.Set("ddmc-city-number", api.address.StationInfo.CityNumber)

		location, _ := api.getLocation()
		header.Set("ddmc-longitude", location[0])
		header.Set("ddmc-latitude", location[1])
	}

	return header, nil
}

func (api *API) newURLEncodedForm() url.Values {
	var params = url.Values{}
	params.Set("uid", `5db2faa481eef77f04ab13e1`)

	params.Set("city_number", `0101`)
	params.Set("api_version", `9.50.0`)
	params.Set("app_version", `2.83.0`)
	params.Set("applet_source", ``)
	params.Set("channel", `applet`)
	params.Set("app_client_id", `4`)
	params.Set("device_token", `WHJMrwNw1k/FKPjcOOgRd+Ed/O2S3GOkz07Wa1UPcfbDL2PfhzepFdBa/QF9u539PLLYm6SKU+84w6mApK0aXmA9Vne9MFdf+dCW1tldyDzmauSxIJm5Txg==1487582755342`)

	// me
	params.Set("sharer_uid", ``)
	params.Set("s_id", `4606726bbe6337d4094e1dec808431d9`)
	params.Set("openid", `osP8I0RgncVIhrJLWwUCb0gi9uDQ`)
	params.Set("h5_source", ``)
	params.Set("time", api.getTime())

	if api.address != nil {
		params.Set("station_id", api.address.StationInfo.ID)
		params.Set("city_number", api.address.StationInfo.CityNumber)
		location, _ := api.getLocation()
		params.Set("longitude", location[0])
		params.Set("latitude", location[1])
	}

	return params
}

func debugMode(debug ...bool) bool {
	return len(debug) > 0 && debug[0]
}

func (api *API) do(req *http.Request, form url.Values, data interface{}, debug ...bool) error {
	if form != nil {
		var m = make(map[string]interface{})
		for k, v := range form {
			m[k] = v[0]
		}

		signResult, err := api.signer.Sign(m)
		if err != nil {
			return err
		}

		form.Set("nars", signResult.Nars)
		form.Set("sesi", signResult.Sesi)

		if debugMode(debug...) {
			log.Println(signResult)
		}

		req.Body = io.NopCloser(strings.NewReader(form.Encode()))
	}

	resp, err := api.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var response Response
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&response); err != nil {
		return err
	}

	if !response.Success {
		log.Println(string(body))
		return NewResponseError(response.Code, response.Message)
	} else if debugMode(debug...) {
		log.Println(string(body))
	}

	return json.Unmarshal(response.Data, data)
}

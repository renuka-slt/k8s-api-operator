/*
 *  Copyright (c) 2020, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 *
 */
package mgw

import (
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	clusterservicev3 "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointservicev3 "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservicev3 "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservicev3 "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	xdsv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"

	"context"
	"flag"
	"fmt"
	logger "github.com/wso2/k8s-api-operator/api-operator/pkg/envoy/loggers"
	apiserver "github.com/wso2/k8s-api-operator/api-operator/pkg/envoy/pkg/apiserver"
	mgwconfig "github.com/wso2/k8s-api-operator/api-operator/pkg/envoy/pkg/configs/confTypes"
	oasParser "github.com/wso2/k8s-api-operator/api-operator/pkg/envoy/pkg/oasparser"
	"google.golang.org/grpc"
	"net"
	"sync/atomic"
)

var (
	debug       bool
	onlyLogging bool

	localhost = "0.0.0.0"

	port        uint
	gatewayPort uint
	alsPort     uint

	mode string

	version int32

	cache cachev3.SnapshotCache
)

const (
	XdsCluster = "xds_cluster"
	Ads        = "ads"
	Xds        = "xds"
	Rest       = "rest"
)

func init() {
	flag.BoolVar(&debug, "debug", true, "Use debug logging")
	flag.BoolVar(&onlyLogging, "onlyLogging", false, "Only demo AccessLogging Service")
	flag.UintVar(&port, "port", 18000, "Management server port")
	flag.UintVar(&gatewayPort, "gateway", 18001, "Management server port for HTTP gateway")
	flag.UintVar(&alsPort, "als", 18090, "Accesslog server port")
	flag.StringVar(&mode, "ads", Ads, "Management server type (ads, xds, rest)")
}

// IDHash uses ID field as the node hash.
type IDHash struct{}

// ID uses the node ID field
func (IDHash) ID(node *corev3.Node) string {
	if node == nil {
		return "unknown"
	}
	return node.Id
}

var _ cachev3.NodeHash = IDHash{}

const grpcMaxConcurrentStreams = 1000000

/**
 * This starts an xDS server at the given port.
 *
 * @param ctx   Context
 * @param server   Xds server instance
 * @param port   Management server port
 */
func RunManagementServer(ctx context.Context, server xdsv3.Server, port uint) {
	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions, grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams))
	grpcServer := grpc.NewServer()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		logger.LoggerMgw.Fatal("failed to listen: ", err)
	}

	// register services
	discoveryv3.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	endpointservicev3.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	clusterservicev3.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	routeservicev3.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	listenerservicev3.RegisterListenerDiscoveryServiceServer(grpcServer, server)

	logger.LoggerMgw.Info("port: ", port, " management server listening")
	//log.Fatalf("", Serve(lis))
	//go func() {
	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			logger.LoggerMgw.Error(err)
		}
	}()
	//<-ctx.Done()
	//grpcServer.GracefulStop()
	//}()

}

/**
 * Recreate the envoy instances from swaggers.
 *
 * @param swaggerCms Swagger config maps
 */
func UpdateEnvoy(swaggerDef *map[string]string) {
	var nodeId string
	if len(cache.GetStatusKeys()) > 0 {
		nodeId = cache.GetStatusKeys()[0]
	}

	listeners, clusters, routes, endpoints := oasParser.GetProductionSources(swaggerDef)

	atomic.AddInt32(&version, 1)
	logger.LoggerMgw.Infof(">>>>>>>>>>>>>>>>>>> creating snapshot Version " + fmt.Sprint(version))
	snap := cachev3.NewSnapshot(fmt.Sprint(version), endpoints, clusters, routes, listeners, nil)
	snap.Consistent()

	err := cache.SetSnapshot(nodeId, snap)
	if err != nil {
		logger.LoggerMgw.Error(err)
	}
}

/**
 * Recreate the envoy instances from swaggers using envoy id.
 *
 * @param location   Swagger files location
 */
func updateEnvoyForSpecificNode(location string, nodeId string) {

	//todo (VirajSalaka): avoid printing the error message
	_, error := cache.GetSnapshot(nodeId)
	//if the snapshot exists, it means that the envoy is already updated.
	//todo: change the logic as move forward
	if error != nil {
		// TODO: CHANGE THIS
		formattedSwaggers := map[string]string{"petstore-swagger": "openapi: \"3.0\"\ninfo:\n  description: \"This is a  sample server Pet-store [https://swagger.o](htp//swag.o)  ssn [.es()s y cn  p  `sie`    n\"\n  version: \"1.0.0\"\n  title: \"Swagger Petstore\"\n\n\nhost: \"petstore.swagger.io:8000\"\nbasePath: \"/api/v3\"\n\nx-wso2-basepath: /petstore/v2\n\nx-wso2-production-endpoints:\n  urls:\n    - 'https://petstore.swagger.io:433/v2/'\n  type: \"https\"\nx-wso2-sandbox-endpoints:\n  urls:\n    - 'http://sandbox.com:8000/api/v3'\n  type: http\n\nservers:\n  - url: 'https://petstore.swagger.io:8000/api/v3'\n\nschemes:\n  - \"https\"\n  - \"http\"\n\npaths:\n  /WSO2Inc:\n    get:\n      x-wso2-production-endpoints:\n        urls:\n          - 'http://www.mocky.io:80/v2/5185415ba171ea3a00704eed'\n        type: https\n      tags:\n        - \"pet\"\n      summary: \"Add a new pet to the store\"\n      description: \"\"\n      operationId: \"addPet\"\n      consumes:\n        - \"application/json\"\n        - \"application/xml\"\n      produces:\n        - \"application/xml\"\n        - \"application/json\"\n      parameters:\n        - in: \"body\"\n          name: \"body\"\n          description: \"Pet object that needs to be added to the store\"\n          required: true\n          schema:\n            $ref: \"#/definitions/Pet\"\n      responses:\n        \"405\":\n          description: \"Invalid input\"\n      security:\n        - petstore_auth:\n            - \"write:pets\"\n            - \"read:pets\"\n\n  /pet/findByStatus:\n    get:\n      tags:\n        - \"pet\"\n      summary: \"Finds Pets by status\"\n      description: \"Multiple status values can be provided with comma separated strings\"\n      operationId: \"findPetsByStatus\"\n      produces:\n        - \"application/xml\"\n        - \"application/json\"\n      parameters:\n        - name: \"status\"\n          in: \"query\"\n          description: \"Status values that need to be considered for filter\"\n          required: true\n          type: \"array\"\n          items:\n            type: \"string\"\n            enum:\n              - \"available\"\n              - \"pending\"\n              - \"sold\"\n            default: \"available\"\n          collectionFormat: \"multi\"\n      responses:\n        \"200\":\n          description: \"successful operation\"\n          schema:\n            type: \"array\"\n            items:\n              $ref: \"#/definitions/Pet\"\n        \"400\":\n          description: \"Invalid status value\"\n      security:\n        - petstore_auth:\n            - \"write:pets\"\n            - \"read:pets\"\n  /pet/{petId}:\n    get:\n      tags:\n        - \"pet\"\n      summary: \"Find pet by ID\"\n      description: \"Returns a single pet\"\n      operationId: \"getPetById\"\n      produces:\n        - \"application/xml\"\n        - \"application/json\"\n      parameters:\n        - name: \"petId\"\n          in: \"path\"\n          description: \"ID of pet to return\"\n          required: true\n          type: \"integer\"\n          format: \"int64\"\n      responses:\n        \"200\":\n          description: \"successful operation\"\n          schema:\n            $ref: \"#/definitions/Pet\"\n        \"400\":\n          description: \"Invalid ID supplied\"\n        \"404\":\n          description: \"Pet not found\"\n      security:\n        - api_key: []\n\n  /data:\n    get:\n      x-wso2-production-endpoints:\n        urls:\n          - 'http://0.0.0.0:5000/base'\n        type: https\n      tags:\n        - \"pet\"\n      summary: \"uploads an image\"\n      description: \"\"\n      operationId: \"uploadFile\"\n      consumes:\n        - \"multipart/form-data\"\n      produces:\n        - \"application/json\"\n      parameters:\n        - name: \"petId\"\n          in: \"path\"\n          description: \"ID of pet to update\"\n          required: true\n          type: \"integer\"\n          format: \"int64\"\n        - name: \"additionalMetadata\"\n          in: \"formData\"\n          description: \"Additional data to pass to server\"\n          required: false\n          type: \"string\"\n        - name: \"file\"\n          in: \"formData\"\n          description: \"file to upload\"\n          required: false\n          type: \"file\"\n      responses:\n        \"200\":\n          description: \"successful operation\"\n          schema:\n            $ref: \"#/definitions/ApiResponse\"\n      security:\n        - petstore_auth:\n            - \"write:pets\"\n            - \"read:pets\"\n  /get-current-statistical:\n    get:\n      x-wso2-production-endpoints:\n        urls:\n          - \"https://www.hpb.health.gov.lk:80/api\"\n        type: http\n      tags:\n        - \"store\"\n      summary: \"Place an order for a pet\"\n\n      description: \"\"\n      operationId: \"placeOrder\"\n      produces:\n        - \"application/xml\"\n        - \"application/json\"\n      parameters:\n        - in: \"body\"\n          name: \"body\"\n          description: \"order placed for purchasing the pet\"\n          required: true\n          schema:\n            $ref: \"#/definitions/Order\"\n      responses:\n        \"200\":\n          description: \"successful operation\"\n          schema:\n            $ref: \"#/definitions/Order\"\n        \"400\":\n          description: \"Invalid Order\"\n  /testResource:\n    get:\n      x-wso2-production-endpoints:\n        urls:\n          - \"https://www.test.lk:80/api\"\n        type: http\n      tags:\n        - \"store\"\n      summary: \"Place an order for a pet\"\n\n      description: \"\"\n      operationId: \"placeOrder\"\n      produces:\n        - \"application/xml\"\n        - \"application/json\"\n      parameters:\n        - in: \"body\"\n          name: \"body\"\n          description: \"order placed for purchasing the pet\"\n          required: true\n          schema:\n            $ref: \"#/definitions/Order\"\n      responses:\n        \"200\":\n          description: \"successful operation\"\n          schema:\n            $ref: \"#/definitions/Order\"\n        \"400\":\n          description: \"Invalid Order\"\nsecurityDefinitions:\n  petstore_auth:\n    type: \"oauth2\"\n    authorizationUrl: \"http://petstore.swagger.io/oauth/dialog\"\n    flow: \"implicit\"\n    scopes:\n      write:pets: \"modify pets in your account\"\n      read:pets: \"read your pets\"\n  api_key:\n    type: \"apiKey\"\n    name: \"api_key\"\n    in: \"header\"\ndefinitions:\n  Order:\n    type: \"object\"\n    properties:\n      id:\n        type: \"integer\"\n        format: \"int64\"\n      petId:\n        type: \"integer\"\n        format: \"int64\"\n      quantity:\n        type: \"integer\"\n        format: \"int32\"\n      shipDate:\n        type: \"string\"\n        format: \"date-time\"\n      status:\n        type: \"string\"\n        description: \"Order Status\"\n        enum:\n          - \"placed\"\n          - \"approved\"\n          - \"delivered\"\n      complete:\n        type: \"boolean\"\n        default: false\n    xml:\n      name: \"Order\"\n  Category:\n    type: \"object\"\n    properties:\n      id:\n        type: \"integer\"\n        format: \"int64\"\n      name:\n        type: \"string\"\n    xml:\n      name: \"Category\"\n  User:\n    type: \"object\"\n    properties:\n      id:\n        type: \"integer\"\n        format: \"int64\"\n      username:\n        type: \"string\"\n      firstName:\n        type: \"string\"\n      lastName:\n        type: \"string\"\n      email:\n        type: \"string\"\n      password:\n        type: \"string\"\n      phone:\n        type: \"string\"\n      userStatus:\n        type: \"integer\"\n        format: \"int32\"\n        description: \"User Status\"\n    xml:\n      name: \"User\"\n  Tag:\n    type: \"object\"\n    properties:\n      id:\n        type: \"integer\"\n        format: \"int64\"\n      name:\n        type: \"string\"\n    xml:\n      name: \"Tag\"\n  Pet:\n    type: \"object\"\n    required:\n      - \"name\"\n      - \"photoUrls\"\n    properties:\n      id:\n        type: \"integer\"\n        format: \"int64\"\n      category:\n        $ref: \"#/definitions/Category\"\n      name:\n        type: \"string\"\n        example: \"doggie\"\n      photoUrls:\n        type: \"array\"\n        xml:\n          name: \"photoUrl\"\n          wrapped: true\n        items:\n          type: \"string\"\n      tags:\n        type: \"array\"\n        xml:\n          name: \"tag\"\n          wrapped: true\n        items:\n          $ref: \"#/definitions/Tag\"\n      status:\n        type: \"string\"\n        description: \"pet status in the store\"\n        enum:\n          - \"available\"\n          - \"pending\"\n          - \"sold\"\n    xml:\n      name: \"Pet\"\n  ApiResponse:\n    type: \"object\"\n    properties:\n      code:\n        type: \"integer\"\n        format: \"int32\"\n      type:\n        type: \"string\"\n      message:\n        type: \"string\"\nexternalDocs:\n  description: \"Find out more about Swagger\"\n  url: \"http://swagger.io\"\n"}
		listeners, clusters, routes, endpoints := oasParser.GetProductionSources(&formattedSwaggers)

		atomic.AddInt32(&version, 1)
		logger.LoggerMgw.Infof(">>>>>>>>>>>>>>>>>>> creating snapshot Version for node " + nodeId + " : " + fmt.Sprint(version))
		snap := cachev3.NewSnapshot(fmt.Sprint(version), endpoints, clusters, routes, listeners, nil)
		snap.Consistent()

		err := cache.SetSnapshot(nodeId, snap)
		if err != nil {
			logger.LoggerMgw.Error(err)
		}
	}
}

/**
 * Run the management grpc server.
 *
 * @param conf  Swagger files location
 */
func Run(conf *mgwconfig.Config, ctx context.Context) {
	//log config watcher
	//watcherLogConf, _ := fsnotify.NewWatcher()
	//errC := watcherLogConf.Add("resources/conf/log_config.toml")
	//
	//if errC != nil {
	//	logger.LoggerMgw.Fatal("Error reading the log configs. ", err)
	//}

	logger.LoggerMgw.Info("Starting control plane ....")

	cache = cachev3.NewSnapshotCache(mode != Ads, IDHash{}, nil)

	//todo: implement own set of callbacks.
	cbv3 := &Callbacks{Signal: nil, Debug: true}
	srv := xdsv3.NewServer(ctx, cache, cbv3)

	//als := &myals.AccessLogService{}
	//go RunAccessLogServer(ctx, als, alsPort)

	// start the xDS server
	RunManagementServer(ctx, srv, port)
	go apiserver.Start(conf)

	//	UpdateEnvoy(conf.Apis.Location)
	//OUTER:
	//	for {
	//		select {
	//		case c := <-watcher.Events:
	//			switch c.Op.String() {
	//			case "WRITE":
	//				logger.LoggerMgw.Info("Loading updated swagger definition...")
	//				UpdateEnvoy(conf.Apis.Location)
	//			}
	//		case l := <-watcherLogConf.Events:
	//			switch l.Op.String() {
	//			case "WRITE":
	//				logger.LoggerMgw.Info("Loading updated log config file...")
	//				configs.ClearLogConfigInstance()
	//				logger.UpdateLoggers()
	//			}
	//		case s := <-sig:
	//			switch s {
	//			case os.Interrupt:
	//				logger.LoggerMgw.Info("Shutting down...")
	//				break OUTER
	//			}
	//		}
	//	}
	//	logger.LoggerMgw.Info("Bye!")
}

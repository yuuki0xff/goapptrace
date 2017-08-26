/// <reference types="angular" />

enum WindowStatus {
    Default = 0,
    Minimize,
    Maximize,
    Moving,
}

enum GraphStatus {}

enum GraphType {}

enum GraphSize {
    Collapsed = 0,
    Expanded,
}

class Graph {
    status: GraphStatus;
    type: GraphType;
    size: GraphSize;
    xaxis: string;
    yaxis: string;
}

class GraphWindow {
    title: string;
    description: string;
    status: WindowStatus;
    graph: Graph;

    constructor() {
        // TODO: 仮の値で初期化
        this.title = "Graph Title";
        this.description = "Description";
        this.status = WindowStatus.Default;
        this.graph = new Graph();

        this.graph.size = GraphSize.Collapsed;
    }
}

let app = angular.module("viewerApp", []);
app.controller("viewerCtl", ($scope) => {
    let numberOfDummyGraphs = 30;
    $scope.GraphSize = GraphSize;
    $scope.WindowStatus = WindowStatus;

    $scope.isEditable = true;
    $scope.graphs = [];
    for (let i = 0; i < numberOfDummyGraphs; i++) {
        $scope.graphs.push(new GraphWindow());
    }
});

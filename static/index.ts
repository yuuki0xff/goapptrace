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

app.directive("svgGraph", ($compile, $sce, $document) => {
    // Query-string付きのURLで発生するエラーを止める
    $sce.RESOURCE_URL = ['self'];

    return {
        restrict: 'E',
        // TODO: templateURLを変更
        templateUrl: "/api/log.svg?width=24000&height=800&layout=goroutine&color-rule=module&colors=6&start=1&scale=1.0",
        link: (scope, element, attrs) => {
            let svgElm: any = element.find("svg");

            // position of mouse pointer last dragging event occurred
            let startX = 0, startY = 0;
            // position of viewBox
            let x = 0, y = 0;
            // viewBox size / viewPort size
            let scale = 1.0;

            $document.bind('mousewheel DOMMouseScroll', mousewheel);
            element.on('mousedown', function (event) {
                // Prevent default dragging of selected content
                event.preventDefault();
                startX = event.pageX;
                startY = event.pageY;
                $document.on('mousemove', mousemove);
                $document.on('mouseup', mouseup);
            });

            function resize() {
                let w, h;
                w = element[0].offsetWidth * scale;
                h = element[0].offsetHeight * scale;
                svgElm.attr('viewBox', [x, y, w, h].join(' '));
            }

            function mousemove(event) {
                // raw moving distance
                let dx = event.pageX - startX;
                let dy = event.pageY - startY;
                startX = event.pageX;
                startY = event.pageY;

                // set viewBox
                x -= dx * scale;
                y -= dy * scale;
                resize();
            }

            function mousewheel(event) {
                let oldw, oldh;
                let w, h;
                oldw = element[0].offsetWidth * scale;
                oldh = element[0].offsetHeight * scale;

                if (event.originalEvent.wheelDelta > 0 || event.originalEvent.detail < 0) {
                    // scroll up
                    scale *= 0.8;
                } else {
                    // scroll down
                    scale *= 1 / 0.8;
                }

                w = element[0].offsetWidth * scale;
                h = element[0].offsetHeight * scale;
                x -= (-oldw + w) / 2;
                y -= (-oldh + h) / 2;
                resize();
                return false
            }

            function mouseup() {
                $document.off('mousemove', mousemove);
                $document.off('mouseup', mouseup);
            }
        }
    };
});

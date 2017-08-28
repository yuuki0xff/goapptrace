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

            // position of started drag.
            let startX = 0, startY = 0;
            // position of viewBox
            let vx = 0, vy = 0;
            // viewBox size / viewPort size
            let scale = 1.0;
            // temporary position of viewBox
            let x, y;

            $document.bind('mousewheel DOMMouseScroll', mousewheel);
            element.on('mousedown', function (event) {
                // Prevent default dragging of selected content
                event.preventDefault();
                startX = event.pageX;
                startY = event.pageY;
                $document.on('mousemove', mousemove);
                $document.on('mouseup', mouseup);
            });

            function mousemove(event) {
                // raw moving distance
                let dx = event.pageX - startX;
                let dy = event.pageY - startY;

                // set viewBox
                let w, h;
                x = vx - dx * scale;
                y = vy - dy * scale;
                w = element[0].offsetWidth * scale;
                h = element[0].offsetHeight * scale;
                svgElm.attr('viewBox', [x, y, w, h].join(' '));
            }

            function mousewheel(event) {
                if (event.target != svgElm[0])
                    return true;

                let w, h;
                if (event.originalEvent.wheelDelta > 0 || event.originalEvent.detail < 0) {
                    // scroll up
                    // console.log('scroll up', event, event.originalEvent.wheelDelta, event.originalEvent.detail);
                    scale *= 0.8;
                    w = element[0].offsetWidth * scale;
                    h = element[0].offsetHeight * scale;
                    x = vx - w / 2;
                    y = vy - h / 2;
                    console.log('scale in', scale);
                } else {
                    // scroll down
                    // console.log('scroll down', event, event.originalEvent.wheelDelta, event.originalEvent.detail)
                    scale *= 1.2;
                    w = element[0].offsetWidth * scale;
                    h = element[0].offsetHeight * scale;
                    x = vx - w / 2;
                    y = vy - h / 2;
                    console.log('scale out', scale)
                }
                svgElm.attr('viewBox', [x, y, w, h].join(' '));
                return false
            }

            function mouseup() {
                // update viewBox position
                vx = x;
                vy = y;

                $document.off('mousemove', mousemove);
                $document.off('mouseup', mouseup);
            }
        }
    };
});

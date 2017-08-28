/// <reference types="angular" />
/// <reference types="jquery" />
/// <reference path="./node_modules/svg.js/svg.js.d.ts" />

// monkey patch for fixes bug that "svg.js" and "svg.js.d.ts".
var SVG: svgjs.Library;

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

let app = angular.module("viewerApp", ["content-editable"]);
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
    let svgWidth = 100000;
    let svgHeight = 100000;

    if (!SVG.supported) {
        alert(`**** ERROR ****

This browser is not supported SVG.
Please open on other browser.`);
    }

    function guid() {
        function s4() {
            return Math.floor((1 + Math.random()) * 0x10000)
                .toString(16)
                .substring(1);
        }

        return s4() + s4() + '-' + s4() + '-' + s4() + '-' +
            s4() + '-' + s4() + s4() + s4();
    }

    return {
        restrict: 'E',
        link: (scope, element, attrs) => {
            let rootElm = $(element[0]);
            let rootId = guid();
            rootElm.attr('id', rootId);

            let svg = SVG(rootId);
            let svgElm = $(svg.native());

            function loadSvg(layout, colorRule, colors, start) {
                $.ajax({
                    'url': '/api/log.svg',
                    'dataType': 'xml',
                    'data': {
                        'width': svgWidth,
                        'height': svgHeight,
                        'layout': layout,
                        'color-rule': colorRule,
                        'colors': colors,
                        'start': start,
                        'scale': 1.0, // it's dummy parameter.
                    },
                }).done((data) => {
                    let newsvg = data.children[0];
                    let gElm = $(svg.element('g').native());
                    $(newsvg.children).appendTo(gElm);
                }).fail(() => {
                    alert('Failed to load svg file.');
                })

            }

            // TODO: 適切なタイミングで、追加のsvgを読み込む
            loadSvg(
                'goroutine',
                'module',
                6,
                1);

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

            function isDescendant(parent, child) {
                let node = child.parentNode;
                while (node) {
                    if (node == parent) {
                        return true;
                    }
                    node = node.parentNode;
                }
                return false;
            }

            // マウスの中ボタンを回転したときのエベントハンドラ。
            // マウスポインタの位置を基準にして、拡大縮小を行う
            function mousewheel(event) {
                if (event.target != svgElm[0] && !isDescendant(svgElm[0], event.target)) {
                    return true;
                }

                let oldScale = scale;

                if (event.originalEvent.wheelDelta > 0 || event.originalEvent.detail < 0) {
                    // scroll up
                    scale *= 0.8;
                } else {
                    // scroll down
                    scale *= 1 / 0.8;
                }

                let crect = element[0].getBoundingClientRect();
                let bodycret = document.body.getBoundingClientRect();
                let elmPageX, elmPageY;
                elmPageX = crect.left - bodycret.left;
                elmPageY = crect.top - bodycret.top;

                let w, h;
                w = crect.width;
                h = crect.height;

                // viewPort上のマウスの座標
                // 0.0 <= (mouseX, mouseY) <= 1.0
                let mouseX, mouseY;
                mouseX = (event.pageX - elmPageX) / w;
                mouseY = (event.pageY - elmPageY) / h;

                // viewBox上の基準座標
                // この位置を起点として拡大縮小を行う。この座標のviewPort上の位置は移動してはいけない。
                let baseX, baseY;
                baseX = x + mouseX * w * oldScale;
                baseY = y + mouseY * h * oldScale;

                x = baseX - mouseX * w * scale;
                y = baseY - mouseY * h * scale;
                resize();
                return false;
            }

            function mouseup() {
                $document.off('mousemove', mousemove);
                $document.off('mouseup', mouseup);
            }
        }
    };
});

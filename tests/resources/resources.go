/*
 * Copyright 2024 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package resources

import _ "embed"

//go:embed script_err.bpmn
var ScriptErrBpmn string

const SvgExample = `<svg height='48' version='1.1' viewBox='167 96 134 48' width='134' xmlns='http://www.w3.org/2000/svg' xmlns:xlink='http://www.w3.org/1999/xlink'><defs><marker id='sequenceflow-end-white-black-3gh21e50i1p8scvqmvrotmp9p' markerHeight='10' markerWidth='10' orient='auto' refX='11' refY='10' viewBox='0 0 20 20'><path d='M 1 5 L 11 10 L 1 15 Z' style='fill: black; stroke-width: 1px; stroke-linecap: round; stroke-dasharray: 10000, 1; stroke: black;'/></marker></defs><g class='djs-group'><g class='djs-element djs-connection' data-element-id='SequenceFlow_04zz9eb' style='display: block;'><g class='djs-visual'><path d='m  209,120L259,120 ' style='fill: none; stroke-width: 2px; stroke: black; stroke-linejoin: round; marker-end: url(&apos;#sequenceflow-end-white-black-3gh21e50i1p8scvqmvrotmp9p&apos;);'/></g><polyline class='djs-hit' points='209,120 259,120 ' style='fill: none; stroke-opacity: 0; stroke: white; stroke-width: 15px;'/><rect class='djs-outline' height='12' style='fill: none;' width='62' x='203' y='114'/></g></g><g class='djs-group'><g class='djs-element djs-shape' data-element-id='StartEvent_1' style='display: block;' transform='translate(173 102)'><g class='djs-visual'><circle cx='18' cy='18' r='18' style='stroke: black; stroke-width: 2px; fill: white; fill-opacity: 0.95;'/><path d='m 8.459999999999999,11.34 l 0,12.6 l 18.900000000000002,0 l 0,-12.6 z l 9.450000000000001,5.4 l 9.450000000000001,-5.4' style='fill: white; stroke-width: 1px; stroke: black;'/></g><rect class='djs-hit' height='36' style='fill: none; stroke-opacity: 0; stroke: white; stroke-width: 15px;' width='36' x='0' y='0'></rect><rect class='djs-outline' height='48' style='fill: none;' width='48' x='-6' y='-6'></rect></g></g><g class='djs-group'><g class='djs-element djs-shape' data-element-id='EndEvent_056p30q' style='display: block;' transform='translate(259 102)'><g class='djs-visual'><circle cx='18' cy='18' r='18' style='stroke: black; stroke-width: 4px; fill: white; fill-opacity: 0.95;'/></g><rect class='djs-hit' height='36' style='fill: none; stroke-opacity: 0; stroke: white; stroke-width: 15px;' width='36' x='0' y='0'></rect><rect class='djs-outline' height='48' style='fill: none;' width='48' x='-6' y='-6'></rect></g></g></svg>`

ng = angular.module 'myApp'

ng.config ($stateProvider, navbarProvider) ->
  $stateProvider.state 'plot',
    url: '/plot'
    templateUrl: '/plot/plot.html'
    controller: plotCtrl
  navbarProvider.add '/plot', 'Plot', 30

plotCtrl = ($scope, jeebus) ->
  drivers = {}

  # handle incoming data for a plot
  handlePlotData = (row, point) ->
    #console.log "PlotHandler:", arguments
    s = $scope.series[row]
    if s?
      s.values ?= []
      s.values.push([point.Asof, point.Avg])
    #console.log "So far:", $scope.series[row]

  # request data for a plot
  fetchPlot = (rowIx) ->
    # if we have info on that reading, then actually fetch the sensor data
    readings = $scope.readings
    if readings.length > rowIx
      row = readings[rowIx]
      $scope.series ?= []
      $scope.series[0] = {
        dbkey: "#{row.loc}/#{row.param}"
        label: "#{row.loc} #{row.param}"
        start: 0
        end: 0
        values: []
      }
      sensor = "sensor/#{$scope.series[0].dbkey}"
      console.log "Fetching plot data for", sensor
      now = (new Date()).getTime()
      jeebus.timeRange sensor, now-3600*1000, now, 20*1000, angular.bind(this, handlePlotData, 0)
        .on 'sync', ->
          updatePlot(0)

  updatePlot = (rowIx) ->
    series = $scope.series[rowIx]
    console.log "Displaying plot", series.label, "with", series.values.length, "points"
    sorter = (d) -> d[0]
    $scope.flotData ?= []
    $scope.flotData[0] = {
        label: series.label
        data: _.sortBy(series.values, sorter)
      }


  # set which readings row we're plotting
  setPlot = (rowIx) ->
    $scope.plotRow = rowIx
    $scope.flotData ?= []
    $scope.flotOptions = {
      xaxis: {
        mode: "time"
      }
    }
    fetchPlot(rowIx)

  rowHandler = (key, row) ->
    # loc: ... val: [c1:12,c2:34,...]
    {loc,ms,val,typ} = row
    for param, raw of val
      id = "#{key} - #{param}" # device id
      @set id, adjust {loc,param,raw,ms,typ}
      if @keys[id] == 0
        fetchPlot(0)
      console.log "set", id, "to", adjust {loc,param,raw,ms,typ}
    #console.log "rh readings=", $scope.readings
    #console.log "rh series=", $scope.series
    #console.log "keys=", @keys
    #console.log "rows=", @rows
    null

  adjust = (row) ->
    row.value = row.raw
    info = drivers.get "#{row.typ}/#{row.param}"
    if info?
      row.param = info.name
      row.unit = info.unit
      # apply some scaling and formatting
      if info.factor
        row.value *= info.factor
      if info.scale < 0
        row.value *= Math.pow 10, -info.scale
      else if info.scale >= 0
        row.value /= Math.pow 10, info.scale
        row.value = row.value.toFixed info.scale
    row

  setup = ->
    drivers = jeebus.attach 'driver'
      .on 'sync', ->
        jeebus.attach 'reading', rowHandler
          .on 'init', ->
            $scope.readings = @rows
            setPlot(0)
      
  setup()  if $scope.serverPlot is 'connected'
  $scope.$on 'ws-open', setup

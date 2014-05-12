ng = angular.module 'myApp'

ng.config ($stateProvider, navbarProvider) ->
  $stateProvider.state 'status',
    url: '/status'
    templateUrl: '/status/status.html'
    controller: statusCtrl
  navbarProvider.add '/status', 'Status', 30

statusCtrl = ($scope, jeebus) ->
  drivers = {}
  
  rowHandler = (key, row) ->
    # loc: ... val: [c1:12,c2:34,...]
    {loc,ms,val,typ} = row
    for param, raw of val
      id = "#{key} - #{param}" # device id
      @put id, adjust {loc,param,raw,ms,typ}

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
      
  setup()  if $scope.serverStatus is 'connected'
  $scope.$on 'ws-open', setup

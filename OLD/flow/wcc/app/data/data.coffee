ng = angular.module 'myApp'

ng.config ($stateProvider, navbarProvider) ->
  $stateProvider.state 'data',
    url: '/data/:table'
    templateUrl: '/data/data.html'
    controller: dataCtrl
  navbarProvider.add '/data/', 'Data', 35

dataCtrl = ($scope, $stateParams, $location, jeebus) ->
  # FIXME: this gets called far too often, and there's no cleanup yet!
  setup = ->
    $scope.tables = jeebus.attach 'table'
      .on 'sync', ->
        $scope.colInfo = @get($scope.table).attr.split ' '
        $scope.columns = jeebus.attach "column/#{$scope.table}"
          .on 'sync', ->
            $scope.data = jeebus.attach($scope.table)
      
  $scope.changeTable = (t) ->
    $scope.cursor = null
    $scope.table = t or 'table'
    $location.path "/data/#{$scope.table}"
    setup()  if $scope.serverStatus is 'connected'

  $scope.editRow = (row) ->
    $scope.cursor = row
    
  $scope.deleteRow = ->
    if $scope.allowDelete and $scope.cursor?
      $scope.allowDelete = false
      console.log 'DELETE', $scope.table, $scope.cursor

  $scope.$on 'ws-open', setup
  $scope.changeTable $stateParams.table

(function() {
  var demoCtrl, ng;

  ng = angular.module('myApp');

  ng.config(function($stateProvider, navbarProvider) {
    $stateProvider.state('demo', {
      url: '/',
      templateUrl: '/demo/demo.html',
      controller: demoCtrl
    });
    return navbarProvider.add('/', 'Demo', 25);
  });

  demoCtrl = function($scope, jeebus) {
    $scope.echoTest = function() {
      jeebus.send("echoTest!");
      return jeebus.rpc('echo', 'Echo', 'me!').then(function(r) {
        return $scope.message = r;
      });
    };
    $scope.dbKeysTest = function() {
      return jeebus.rpc('db-keys', '/reading/').then(function(r) {
        return $scope.message = r;
      });
    };
    return $scope.mqttTest = function() {
      return jeebus.gadget('MQTTSub', {
        Topic: '/reading/#',
        Port: ':1883'
      }).on('Out', function(r) {
        return $scope.message = r;
      });
    };
  };

}).call(this);

//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiZGVtby5jb2ZmZWUiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUE7QUFBQSxNQUFBLFlBQUE7O0FBQUEsRUFBQSxFQUFBLEdBQUssT0FBTyxDQUFDLE1BQVIsQ0FBZSxPQUFmLENBQUwsQ0FBQTs7QUFBQSxFQUVBLEVBQUUsQ0FBQyxNQUFILENBQVUsU0FBQyxjQUFELEVBQWlCLGNBQWpCLEdBQUE7QUFDUixJQUFBLGNBQWMsQ0FBQyxLQUFmLENBQXFCLE1BQXJCLEVBQ0U7QUFBQSxNQUFBLEdBQUEsRUFBSyxHQUFMO0FBQUEsTUFDQSxXQUFBLEVBQWEsaUJBRGI7QUFBQSxNQUVBLFVBQUEsRUFBWSxRQUZaO0tBREYsQ0FBQSxDQUFBO1dBSUEsY0FBYyxDQUFDLEdBQWYsQ0FBbUIsR0FBbkIsRUFBd0IsTUFBeEIsRUFBZ0MsRUFBaEMsRUFMUTtFQUFBLENBQVYsQ0FGQSxDQUFBOztBQUFBLEVBU0EsUUFBQSxHQUFXLFNBQUMsTUFBRCxFQUFTLE1BQVQsR0FBQTtBQUVULElBQUEsTUFBTSxDQUFDLFFBQVAsR0FBa0IsU0FBQSxHQUFBO0FBQ2hCLE1BQUEsTUFBTSxDQUFDLElBQVAsQ0FBWSxXQUFaLENBQUEsQ0FBQTthQUNBLE1BQU0sQ0FBQyxHQUFQLENBQVcsTUFBWCxFQUFtQixNQUFuQixFQUEyQixLQUEzQixDQUNFLENBQUMsSUFESCxDQUNRLFNBQUMsQ0FBRCxHQUFBO2VBQ0osTUFBTSxDQUFDLE9BQVAsR0FBaUIsRUFEYjtNQUFBLENBRFIsRUFGZ0I7SUFBQSxDQUFsQixDQUFBO0FBQUEsSUFNQSxNQUFNLENBQUMsVUFBUCxHQUFvQixTQUFBLEdBQUE7YUFDbEIsTUFBTSxDQUFDLEdBQVAsQ0FBVyxTQUFYLEVBQXNCLFdBQXRCLENBQ0UsQ0FBQyxJQURILENBQ1EsU0FBQyxDQUFELEdBQUE7ZUFDSixNQUFNLENBQUMsT0FBUCxHQUFpQixFQURiO01BQUEsQ0FEUixFQURrQjtJQUFBLENBTnBCLENBQUE7V0FXQSxNQUFNLENBQUMsUUFBUCxHQUFrQixTQUFBLEdBQUE7YUFDaEIsTUFBTSxDQUFDLE1BQVAsQ0FBYyxTQUFkLEVBQXlCO0FBQUEsUUFBQSxLQUFBLEVBQU8sWUFBUDtBQUFBLFFBQXFCLElBQUEsRUFBTSxPQUEzQjtPQUF6QixDQUNFLENBQUMsRUFESCxDQUNNLEtBRE4sRUFDYSxTQUFDLENBQUQsR0FBQTtlQUNULE1BQU0sQ0FBQyxPQUFQLEdBQWlCLEVBRFI7TUFBQSxDQURiLEVBRGdCO0lBQUEsRUFiVDtFQUFBLENBVFgsQ0FBQTtBQUFBIiwic291cmNlc0NvbnRlbnQiOlsibmcgPSBhbmd1bGFyLm1vZHVsZSAnbXlBcHAnXG5cbm5nLmNvbmZpZyAoJHN0YXRlUHJvdmlkZXIsIG5hdmJhclByb3ZpZGVyKSAtPlxuICAkc3RhdGVQcm92aWRlci5zdGF0ZSAnZGVtbycsXG4gICAgdXJsOiAnLydcbiAgICB0ZW1wbGF0ZVVybDogJy9kZW1vL2RlbW8uaHRtbCdcbiAgICBjb250cm9sbGVyOiBkZW1vQ3RybFxuICBuYXZiYXJQcm92aWRlci5hZGQgJy8nLCAnRGVtbycsIDI1XG5cbmRlbW9DdHJsID0gKCRzY29wZSwgamVlYnVzKSAtPlxuXG4gICRzY29wZS5lY2hvVGVzdCA9IC0+XG4gICAgamVlYnVzLnNlbmQgXCJlY2hvVGVzdCFcIiAjIHNlbmQgYSB0ZXN0IG1lc3NhZ2UgdG8gSkIgc2VydmVyJ3Mgc3Rkb3V0XG4gICAgamVlYnVzLnJwYyAnZWNobycsICdFY2hvJywgJ21lISdcbiAgICAgIC50aGVuIChyKSAtPlxuICAgICAgICAkc2NvcGUubWVzc2FnZSA9IHJcblxuICAkc2NvcGUuZGJLZXlzVGVzdCA9IC0+XG4gICAgamVlYnVzLnJwYyAnZGIta2V5cycsICcvcmVhZGluZy8nXG4gICAgICAudGhlbiAocikgLT5cbiAgICAgICAgJHNjb3BlLm1lc3NhZ2UgPSByXG5cbiAgJHNjb3BlLm1xdHRUZXN0ID0gLT5cbiAgICBqZWVidXMuZ2FkZ2V0ICdNUVRUU3ViJywgVG9waWM6ICcvcmVhZGluZy8jJywgUG9ydDogJzoxODgzJ1xuICAgICAgLm9uICdPdXQnLCAocikgLT5cbiAgICAgICAgJHNjb3BlLm1lc3NhZ2UgPSByXG4iXX0=

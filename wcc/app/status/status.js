(function() {
  var ng, statusCtrl;

  ng = angular.module('myApp');

  ng.config(function($stateProvider, navbarProvider) {
    $stateProvider.state('status', {
      url: '/status',
      templateUrl: '/status/status.html',
      controller: statusCtrl
    });
    return navbarProvider.add('/status', 'Status', 30);
  });

  statusCtrl = function($scope, jeebus) {
    var adjust, drivers, rowHandler, setup;
    drivers = {};
    rowHandler = function(key, row) {
      var id, loc, ms, param, raw, typ, val, _results;
      loc = row.loc, ms = row.ms, val = row.val, typ = row.typ;
      _results = [];
      for (param in val) {
        raw = val[param];
        id = "" + key + " - " + param;
        _results.push(this.put(id, adjust({
          loc: loc,
          param: param,
          raw: raw,
          ms: ms,
          typ: typ
        })));
      }
      return _results;
    };
    adjust = function(row) {
      var info;
      row.value = row.raw;
      info = drivers.get("" + row.typ + "/" + row.param);
      if (info != null) {
        row.param = info.name;
        row.unit = info.unit;
        if (info.factor) {
          row.value *= info.factor;
        }
        if (info.scale < 0) {
          row.value *= Math.pow(10, -info.scale);
        } else if (info.scale >= 0) {
          row.value /= Math.pow(10, info.scale);
          row.value = row.value.toFixed(info.scale);
        }
      }
      return row;
    };
    setup = function() {
      return drivers = jeebus.attach('driver').on('sync', function() {
        return jeebus.attach('reading', rowHandler).on('init', function() {
          return $scope.readings = this.rows;
        });
      });
    };
    if ($scope.serverStatus === 'connected') {
      setup();
    }
    return $scope.$on('ws-open', setup);
  };

}).call(this);

//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsic3RhdHVzLmNvZmZlZSJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQTtBQUFBLE1BQUEsY0FBQTs7QUFBQSxFQUFBLEVBQUEsR0FBSyxPQUFPLENBQUMsTUFBUixDQUFlLE9BQWYsQ0FBTCxDQUFBOztBQUFBLEVBRUEsRUFBRSxDQUFDLE1BQUgsQ0FBVSxTQUFDLGNBQUQsRUFBaUIsY0FBakIsR0FBQTtBQUNSLElBQUEsY0FBYyxDQUFDLEtBQWYsQ0FBcUIsUUFBckIsRUFDRTtBQUFBLE1BQUEsR0FBQSxFQUFLLFNBQUw7QUFBQSxNQUNBLFdBQUEsRUFBYSxxQkFEYjtBQUFBLE1BRUEsVUFBQSxFQUFZLFVBRlo7S0FERixDQUFBLENBQUE7V0FJQSxjQUFjLENBQUMsR0FBZixDQUFtQixTQUFuQixFQUE4QixRQUE5QixFQUF3QyxFQUF4QyxFQUxRO0VBQUEsQ0FBVixDQUZBLENBQUE7O0FBQUEsRUFTQSxVQUFBLEdBQWEsU0FBQyxNQUFELEVBQVMsTUFBVCxHQUFBO0FBQ1gsUUFBQSxrQ0FBQTtBQUFBLElBQUEsT0FBQSxHQUFVLEVBQVYsQ0FBQTtBQUFBLElBRUEsVUFBQSxHQUFhLFNBQUMsR0FBRCxFQUFNLEdBQU4sR0FBQTtBQUVYLFVBQUEsMkNBQUE7QUFBQSxNQUFDLFVBQUEsR0FBRCxFQUFLLFNBQUEsRUFBTCxFQUFRLFVBQUEsR0FBUixFQUFZLFVBQUEsR0FBWixDQUFBO0FBQ0E7V0FBQSxZQUFBO3lCQUFBO0FBQ0UsUUFBQSxFQUFBLEdBQUssRUFBQSxHQUFFLEdBQUYsR0FBTyxLQUFQLEdBQVcsS0FBaEIsQ0FBQTtBQUFBLHNCQUNBLElBQUMsQ0FBQSxHQUFELENBQUssRUFBTCxFQUFTLE1BQUEsQ0FBTztBQUFBLFVBQUMsS0FBQSxHQUFEO0FBQUEsVUFBSyxPQUFBLEtBQUw7QUFBQSxVQUFXLEtBQUEsR0FBWDtBQUFBLFVBQWUsSUFBQSxFQUFmO0FBQUEsVUFBa0IsS0FBQSxHQUFsQjtTQUFQLENBQVQsRUFEQSxDQURGO0FBQUE7c0JBSFc7SUFBQSxDQUZiLENBQUE7QUFBQSxJQVNBLE1BQUEsR0FBUyxTQUFDLEdBQUQsR0FBQTtBQUNQLFVBQUEsSUFBQTtBQUFBLE1BQUEsR0FBRyxDQUFDLEtBQUosR0FBWSxHQUFHLENBQUMsR0FBaEIsQ0FBQTtBQUFBLE1BQ0EsSUFBQSxHQUFPLE9BQU8sQ0FBQyxHQUFSLENBQVksRUFBQSxHQUFFLEdBQUcsQ0FBQyxHQUFOLEdBQVcsR0FBWCxHQUFhLEdBQUcsQ0FBQyxLQUE3QixDQURQLENBQUE7QUFFQSxNQUFBLElBQUcsWUFBSDtBQUNFLFFBQUEsR0FBRyxDQUFDLEtBQUosR0FBWSxJQUFJLENBQUMsSUFBakIsQ0FBQTtBQUFBLFFBQ0EsR0FBRyxDQUFDLElBQUosR0FBVyxJQUFJLENBQUMsSUFEaEIsQ0FBQTtBQUdBLFFBQUEsSUFBRyxJQUFJLENBQUMsTUFBUjtBQUNFLFVBQUEsR0FBRyxDQUFDLEtBQUosSUFBYSxJQUFJLENBQUMsTUFBbEIsQ0FERjtTQUhBO0FBS0EsUUFBQSxJQUFHLElBQUksQ0FBQyxLQUFMLEdBQWEsQ0FBaEI7QUFDRSxVQUFBLEdBQUcsQ0FBQyxLQUFKLElBQWEsSUFBSSxDQUFDLEdBQUwsQ0FBUyxFQUFULEVBQWEsQ0FBQSxJQUFLLENBQUMsS0FBbkIsQ0FBYixDQURGO1NBQUEsTUFFSyxJQUFHLElBQUksQ0FBQyxLQUFMLElBQWMsQ0FBakI7QUFDSCxVQUFBLEdBQUcsQ0FBQyxLQUFKLElBQWEsSUFBSSxDQUFDLEdBQUwsQ0FBUyxFQUFULEVBQWEsSUFBSSxDQUFDLEtBQWxCLENBQWIsQ0FBQTtBQUFBLFVBQ0EsR0FBRyxDQUFDLEtBQUosR0FBWSxHQUFHLENBQUMsS0FBSyxDQUFDLE9BQVYsQ0FBa0IsSUFBSSxDQUFDLEtBQXZCLENBRFosQ0FERztTQVJQO09BRkE7YUFhQSxJQWRPO0lBQUEsQ0FUVCxDQUFBO0FBQUEsSUF5QkEsS0FBQSxHQUFRLFNBQUEsR0FBQTthQUNOLE9BQUEsR0FBVSxNQUFNLENBQUMsTUFBUCxDQUFjLFFBQWQsQ0FDUixDQUFDLEVBRE8sQ0FDSixNQURJLEVBQ0ksU0FBQSxHQUFBO2VBQ1YsTUFBTSxDQUFDLE1BQVAsQ0FBYyxTQUFkLEVBQXlCLFVBQXpCLENBQ0UsQ0FBQyxFQURILENBQ00sTUFETixFQUNjLFNBQUEsR0FBQTtpQkFDVixNQUFNLENBQUMsUUFBUCxHQUFrQixJQUFDLENBQUEsS0FEVDtRQUFBLENBRGQsRUFEVTtNQUFBLENBREosRUFESjtJQUFBLENBekJSLENBQUE7QUFnQ0EsSUFBQSxJQUFZLE1BQU0sQ0FBQyxZQUFQLEtBQXVCLFdBQW5DO0FBQUEsTUFBQSxLQUFBLENBQUEsQ0FBQSxDQUFBO0tBaENBO1dBaUNBLE1BQU0sQ0FBQyxHQUFQLENBQVcsU0FBWCxFQUFzQixLQUF0QixFQWxDVztFQUFBLENBVGIsQ0FBQTtBQUFBIiwic291cmNlc0NvbnRlbnQiOlsibmcgPSBhbmd1bGFyLm1vZHVsZSAnbXlBcHAnXG5cbm5nLmNvbmZpZyAoJHN0YXRlUHJvdmlkZXIsIG5hdmJhclByb3ZpZGVyKSAtPlxuICAkc3RhdGVQcm92aWRlci5zdGF0ZSAnc3RhdHVzJyxcbiAgICB1cmw6ICcvc3RhdHVzJ1xuICAgIHRlbXBsYXRlVXJsOiAnL3N0YXR1cy9zdGF0dXMuaHRtbCdcbiAgICBjb250cm9sbGVyOiBzdGF0dXNDdHJsXG4gIG5hdmJhclByb3ZpZGVyLmFkZCAnL3N0YXR1cycsICdTdGF0dXMnLCAzMFxuXG5zdGF0dXNDdHJsID0gKCRzY29wZSwgamVlYnVzKSAtPlxuICBkcml2ZXJzID0ge31cbiAgXG4gIHJvd0hhbmRsZXIgPSAoa2V5LCByb3cpIC0+XG4gICAgIyBsb2M6IC4uLiB2YWw6IFtjMToxMixjMjozNCwuLi5dXG4gICAge2xvYyxtcyx2YWwsdHlwfSA9IHJvd1xuICAgIGZvciBwYXJhbSwgcmF3IG9mIHZhbFxuICAgICAgaWQgPSBcIiN7a2V5fSAtICN7cGFyYW19XCIgIyBkZXZpY2UgaWRcbiAgICAgIEBwdXQgaWQsIGFkanVzdCB7bG9jLHBhcmFtLHJhdyxtcyx0eXB9XG5cbiAgYWRqdXN0ID0gKHJvdykgLT5cbiAgICByb3cudmFsdWUgPSByb3cucmF3XG4gICAgaW5mbyA9IGRyaXZlcnMuZ2V0IFwiI3tyb3cudHlwfS8je3Jvdy5wYXJhbX1cIlxuICAgIGlmIGluZm8/XG4gICAgICByb3cucGFyYW0gPSBpbmZvLm5hbWVcbiAgICAgIHJvdy51bml0ID0gaW5mby51bml0XG4gICAgICAjIGFwcGx5IHNvbWUgc2NhbGluZyBhbmQgZm9ybWF0dGluZ1xuICAgICAgaWYgaW5mby5mYWN0b3JcbiAgICAgICAgcm93LnZhbHVlICo9IGluZm8uZmFjdG9yXG4gICAgICBpZiBpbmZvLnNjYWxlIDwgMFxuICAgICAgICByb3cudmFsdWUgKj0gTWF0aC5wb3cgMTAsIC1pbmZvLnNjYWxlXG4gICAgICBlbHNlIGlmIGluZm8uc2NhbGUgPj0gMFxuICAgICAgICByb3cudmFsdWUgLz0gTWF0aC5wb3cgMTAsIGluZm8uc2NhbGVcbiAgICAgICAgcm93LnZhbHVlID0gcm93LnZhbHVlLnRvRml4ZWQgaW5mby5zY2FsZVxuICAgIHJvd1xuXG4gIHNldHVwID0gLT5cbiAgICBkcml2ZXJzID0gamVlYnVzLmF0dGFjaCAnZHJpdmVyJ1xuICAgICAgLm9uICdzeW5jJywgLT5cbiAgICAgICAgamVlYnVzLmF0dGFjaCAncmVhZGluZycsIHJvd0hhbmRsZXJcbiAgICAgICAgICAub24gJ2luaXQnLCAtPlxuICAgICAgICAgICAgJHNjb3BlLnJlYWRpbmdzID0gQHJvd3NcbiAgICAgIFxuICBzZXR1cCgpICBpZiAkc2NvcGUuc2VydmVyU3RhdHVzIGlzICdjb25uZWN0ZWQnXG4gICRzY29wZS4kb24gJ3dzLW9wZW4nLCBzZXR1cFxuIl19

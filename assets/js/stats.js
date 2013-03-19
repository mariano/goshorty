(function($) {
	var parseValues = function(data) {
		var values = [ ["", "Hits"] ];
		for (var i=0, limit=data.length; i < limit; i++) {
			values.push([ data[i].Name, data[i].Value ]);
		}
		return values;
	};

	var loadHits = function(href) {
		var index = href.indexOf("#"),
			what = index >= 0 ? href.substring(index + 1) : null;
		if (!what) {
			what = "day";
		}

		var url = $("#stats").attr("rel");
		if (!url) {
			return;
		}

		var chart = new google.visualization.LineChart($('#hitsChart').get(0));
		$.ajax({
			type: "GET",
			dataType: "json",
			url: url.replace(/\/day$/, "/" + what),
			success: function(data) {
				var maxValue = 0,
					values = parseValues(data);
				for (var i=0, limit=values.length; i < limit; i++) {
					if (values[i][1] > 0 && values[i][1] > maxValue) {
						maxValue = values[i][1];
					}
				}
				chart.draw(google.visualization.arrayToDataTable(values), {
					"vAxis": {"viewWindowMode": "explicit", "viewWindow": { "min": 0 }, "format": maxValue >= 3 ? "#" : "#.#"},
					"legend": {"position": "none"}
				});
			}
		});
	};

	var loadCountries = function() {
		var url = $("#stats").attr("rel");
		if (!url) {
			return;
		}

		var chart = new google.visualization.GeoChart($('#countriesChart').get(0));
		$.ajax({
			type: "GET",
			dataType: "json",
			url: url.replace(/\/day$/, "/countries"),
			success: function(data) {
				var maxValue = 0,
					values = parseValues(data);
				for (var i=0, limit=values.length; i < limit; i++) {
					if (values[i][1] > 0 && values[i][1] > maxValue) {
						maxValue = values[i][1];
					}
				}
				chart.draw(google.visualization.arrayToDataTable(values), {
					"colorAxis": {"colors": ['red','#004411']}
				});
			}
		});
	};

	var loadBrowsers = function() {
		var url = $("#stats").attr("rel");
		if (!url) {
			return;
		}

		var chart = new google.visualization.PieChart($('#browsersChart').get(0));
		$.ajax({
			type: "GET",
			dataType: "json",
			url: url.replace(/\/day$/, "/browsers"),
			success: function(data) {
				var maxValue = 0,
					values = parseValues(data);
				for (var i=0, limit=values.length; i < limit; i++) {
					if (values[i][1] > 0 && values[i][1] > maxValue) {
						maxValue = values[i][1];
					}
				}
				chart.draw(google.visualization.arrayToDataTable(values), {
					"legend": {"position": "bottom"}
				});
			}
		});
	};

	var loadOS = function() {
		var url = $("#stats").attr("rel");
		if (!url) {
			return;
		}

		var chart = new google.visualization.PieChart($('#osChart').get(0));
		$.ajax({
			type: "GET",
			dataType: "json",
			url: url.replace(/\/day$/, "/os"),
			success: function(data) {
				var maxValue = 0,
					values = parseValues(data);
				for (var i=0, limit=values.length; i < limit; i++) {
					if (values[i][1] > 0 && values[i][1] > maxValue) {
						maxValue = values[i][1];
					}
				}
				chart.draw(google.visualization.arrayToDataTable(values), {
					"legend": {"position": "bottom"}
				});
			}
		});
	};

	var loadReferrers = function() {
		var url = $("#stats").attr("rel");
		if (!url) {
			return;
		}

		var chart = new google.visualization.ColumnChart($('#referrersChart').get(0));
		$.ajax({
			type: "GET",
			dataType: "json",
			url: url.replace(/\/day$/, "/referrers"),
			success: function(data) {
				var maxValue = 0,
					values = parseValues(data);
				for (var i=0, limit=values.length; i < limit; i++) {
					if (values[i][1] > 0 && values[i][1] > maxValue) {
						maxValue = values[i][1];
					}
				}
				chart.draw(google.visualization.arrayToDataTable(values), {
					"legend": {"position": "none"}
				});
			}
		});
	};

	google.load("visualization", "1", {packages:["corechart", "geochart"]});
	google.setOnLoadCallback(function() {
		$(function() {
			$("#change a").click(function(e) {
				e.preventDefault();
				loadHits($(this).attr("href"));
			});

			loadHits(location.href);
			loadBrowsers();
			loadCountries();
			loadOS();
			loadReferrers();
		});
	});
})(jQuery);

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

	var loadSources = function() {
		var url = $("#stats").attr("rel");
		if (!url) {
			return;
		}

		$.ajax({
			type: "GET",
			dataType: "json",
			url: url.replace(/\/day$/, "/sources"),
			success: function(data) {
				if (!data.Browsers || !data.Countries || !data.OS || !data.Referrers) {
					return;
				}

				var charts = {
					browsers: new google.visualization.PieChart($('#browsersChart').get(0)),
					countries: new google.visualization.GeoChart($('#countriesChart').get(0)),
					os: new google.visualization.PieChart($('#osChart').get(0)),
					referrers: new google.visualization.ColumnChart($('#referrersChart').get(0))
				};

				charts.browsers.draw(google.visualization.arrayToDataTable(parseValues(data.Browsers)), {
					"legend": {"position": "bottom"}
				});
				charts.countries.draw(google.visualization.arrayToDataTable(parseValues(data.Countries)), {
					"colorAxis": {"colors": ['red','#004411']}
				});
				charts.os.draw(google.visualization.arrayToDataTable(parseValues(data.OS)), {
					"legend": {"position": "bottom"}
				});
				charts.referrers.draw(google.visualization.arrayToDataTable(parseValues(data.Referrers)), {
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
			loadSources();
		});
	});
})(jQuery);

(function($) {
	var load = function(href) {
		var index = href.indexOf("#"),
			what = index >= 0 ? href.substring(index + 1) : null;
		if (!what) {
			what = "day";
		}

		var url = $("#stats").attr("rel");
		if (!url) {
			return;
		}

		var chart = new google.visualization.LineChart($('#chart').get(0));

		$.ajax({
			type: "GET",
			dataType: "json",
			url: url.replace(/\/day$/, "/" + what),
			success: function(data) {
				var values = [ ["", "Hits"] ],
					maxValue = 0;
				for (var i=0, limit=data.length; i < limit; i++) {
					if (data[i].Value > maxValue) {
						maxValue = data[i].Value;
					}
					values.push([ data[i].Name, data[i].Value ]);
				}
				chart.draw(google.visualization.arrayToDataTable(values), {
					"vAxis": {"viewWindowMode": "explicit", "viewWindow": { "min": 0 }, "format": maxValue >= 3 ? "#" : "#.#"},
					"legend": {"position": "none"}
				});
			}
		});
	};

	google.load("visualization", "1", {packages:["corechart"]});
	google.setOnLoadCallback(function() {
		$(function() {
			$("#change a").click(function(e) {
				e.preventDefault();
				load($(this).attr("href"));
			});

			load(location.href);
		});
	});
})(jQuery);

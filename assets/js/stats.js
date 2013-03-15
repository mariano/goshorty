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
				var values = [];
				for (var i=0, limit=data.length; i < limit; i++) {
					values.push([ data[i].Name, data[i].Value ]);
				}
				console.log(values);
				chart.draw(google.visualization.arrayToDataTable(values), {
					"vAxis.minValue": 0,
					"titlePosition": "none"
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

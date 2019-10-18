/*
  change your API host here
  we use convert python third-party module to convert our array-like data to 
  matrix which is needed for heatmap visualizer
*/

const tickDataAPIPrefix = 'http://127.0.0.1:8000/keyvis?start=-60m&tag='
var rawInfo,
  allRanges = []
var heatmapType = 'written_bytes'

let ctime = 0,
  switches = {}

/* 
  leave it alone, or set it up by yourself
  install the requirements.txt by pip, and run `python server`
*/
const convertAPI = 'http://106.75.91.214/convert'

function getData(type) {
  return fetch(tickDataAPIPrefix + type)
    .then(res => res.json())
    .then(json => {
      rawInfo = json
      try {
        const tl =
          '\t' +
          '\t\t\t' +
          json.heatmaps[0].values[0]
            .map((i, idx) => {
              return idx + 'm'
              // return 'time-' + idx
            })
            .join('\t')

        let dlines = [],
          b_num = 0
        json.heatmaps.forEach(h => {
          h.ranges.forEach((i, idx) => {
            const ds = h.labels[2] ? 'index ' + h.labels[2] : 'Data'
            dlines.push(
              [
                `Bucket: bucket-${b_num}`,
                'DB: ' + h.labels[0],
                'Table: ' + h.labels[1],
                'Data Type: ' + ds, //? h.labels[2] : 'DATA',
                ...h.values[idx].map(i => i + 1)
              ].join('\t')
            )
            b_num += 1
          })
        })

        return tl + '\n' + dlines.join('\n')
      } catch (e) {}
    })
}

var about_string = ''

function make_clust(type) {
  $.busyLoadFull('show')

  getData(type).then(data => {
    fetch(convertAPI, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        data //: write(60, 256)
      })
    })
      .then(res => res.json())
      .then(json => {
        console.log(json)

        let network_data = json

        var args = {
          opacity_scale: 'log',
          root: '#container-id-1',
          network_data: network_data,
          about: about_string,
          sidebar_width: 0,
          use_sidebar: false,
          // 'ini_view':{'N_row_var':20}
          ini_expand: true,
          make_row_tooltip_handler: d => {
            const x = rawInfo[0]['buckets'][d.rank]
            return `from ${x['start']} to ${x['end']}`
          },
          make_col_tooltip_handler: d => {
            return ''
            return rawInfo[d.rank].time
          },
          matrix_tip_str_handler: (d, inst_value) => {
            // const len = rawInfo.heatmaps[0].values.length
            // const hidx = Math.floor(d.pos_y / len)
            // const valIdx = d.pos_y % len
            // const x = rawInfo.heatmaps[hidx].ranges[valIdx]
            const x = allRanges[d.pos_y]

            // const x = rawInfo[0]['buckets'][d.pos_y]
            const row_name = `key range from ${JSON.stringify(
              x['start']
            )} to ${JSON.stringify(x['end'])}`
            const col_name = '' // rawInfo[d.pos_x].time
            tooltip_string =
              '<p>' +
              row_name +
              ' and at time ' +
              col_name +
              '</p>' +
              '<div> value: ' +
              inst_value +
              '</div>'
            return tooltip_string
          }
        }
        allRanges = []
        rawInfo.heatmaps.forEach(h => {
          allRanges = [...allRanges, ...h.ranges]
        })

        resize_container(args)

        d3.select(window).on('resize', function() {
          resize_container(args)
          cgm.resize_viz()
        })

        cgm = Heatmap(args)

        // check_setup_enrichr(cgm)

        d3.select(cgm.params.root + ' .wait_message').remove()

        $.busyLoadFull('hide')

        $('.wait_message').html('Heatmap for ' + heatmapType)
      })
  })

  // d3.json('json/' + inst_network, function(network_data) {
  //   // define arguments object

  // })
}

function resize_container(args) {
  var screen_width = window.innerWidth
  var screen_height = window.innerHeight - 20

  d3.select(args.root)
    .style('width', screen_width + 'px')
    .style('height', screen_height + 'px')
}

make_clust(heatmapType)

$('#x-switch-menu').click(e => {
  // $('#svg_container-id-1').remove()
  // $('#svg_container-id-1').busyLoad('show')
  //   const canvas = $('.')
  //   const context = canvas.getContext('2d');

  // context.clearRect(0, 0, canvas.width, canvas.height);
  const t = $(e.target.parentElement).data('fetch-label')
  heatmapType = t
  make_clust(t)
})

// write func to generate three types data matrixs for visualization
function write(times, col) {
  const timesArr = Array(times)
    .fill(1)
    .map((i, u) => 'time' + u)
  const colArr = Array(col) // Math.pow(10, col)
    .fill(1)
    .map((i, u) => 'col-' + u)

  switches[0] = (key, kidx) => {
    return (
      key +
      '\t' +
      timesArr
        .map(i => {
          // 1. const num = Math.ceil(Math.random() * 100000)
          const num =
            (+i.replace('time', '') % 6 === 0 ? 1 : 0) * 100000 +
            Math.ceil(Math.random() * 100000)
          return num
        })
        .join('\t')
    )
  }

  switches[1] = (key, kidx) => {
    return (
      key +
      '\t' +
      timesArr
        .map(i => {
          const timenum = +i.replace('time', '')
          const num =
            (((timenum > 10 && timenum < 45 && kidx < 100 && kidx > 60
              ? 1
              : 0) *
              100000 *
              (timenum - 10)) /
              45) *
              0.9 +
            Math.ceil(Math.random() * 100000) / 2
          return num
        })
        .join('\t')
    )
  }

  switches[2] = (key, kidx) => {
    return (
      key +
      '\t' +
      timesArr
        .map(i => {
          const timenum = +i.replace('time', '')
          const num =
            (timenum > 10 && timenum < 45 && kidx < 240 && kidx > 30 ? 1 : 0) *
              200000 *
              ((kidx - 30) / 180) *
              ((timenum - 3) / 45) *
              0.9 +
            Math.ceil(Math.random() * 100000) / 2
          return num
        })
        .join('\t')
    )
  }

  const ret = '\t' + timesArr.join('\t') + '\n'
  const valRet = colArr.map(switches[ctime % 3]).join('\n')
  ctime += 1
  return ret + valRet
}

def is_true: . | IN(["true", "1", 1, true][]);
(
    .response_body
    | fromjson
)
| .data.productDetail.items
| map ({
    id,
    updated_at,
    created_at,
    name, 
    sku, 
    labels: .promo_label,
    price: .price.regularPrice.amount.value,
    active: (.active | is_true),
    is_in_stock: (.is_in_stock | is_true),
    is_prescription: (.rec_need | is_true),
    delivery: (.delivery | is_true),
    thermolabile: (.thermolabile | is_true),
    groups: (
        .breadcrumbs
        | fromjson
        | map (.path)
        | flatten
        | unique
        | map ({id, name})
    ),
    forms: (
        .grouped_products
        | map ({
            sku, 
            name: .form_vypysk_df, 
            values: {
                numero: .fasov_chisl_df, 
                measure: .dozir_chisl_df
            }, 
            texts: {
                numero: .fasov_ap_otkor_df, 
                measure: .dozir_ap_otkor_df
            }
        })
    ),
    manufacturer: {
        id: .manufacturer_id.option_id,
        name: .manufacturer_id.label,
        ru: .manufacturer_ru.label
    },
    images: (
        .media_gallery
        | map ({
            main: .url_image,
            thumbnail: .url_thumbnail,
            small: .url_small_image,
        })
    ),
    mnn: {
        ru: .mnn_ru,
        all: (
            .mnn_id
            | map ({
                id: .option_id,
                value: .label
            })
        )
    },
    attributes: (
        [
            .specification_set_attributes,
            .description_set_attributes
        ]
        | flatten
        | unique
        | map ({
            name: .attribute_label,
            values: (
                .values
                |map(.value)
            )
        })
    )
})
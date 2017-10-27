port module App exposing (..)

{-| TodoMVC implemented in Elm, using plain HTML and CSS for rendering.

This application is broken up into three key parts:

1.  Model - a full definition of the application's state
2.  Update - a way to step the application state forward
3.  View - a way to visualize our application state with HTML

This clean division of concerns is a core part of Elm. You can read more about
this in <http://guide.elm-lang.org/architecture/index.html>

-}

import Dom
import Html exposing (..)
import Html.Attributes exposing (..)
import Html.Events exposing (..)
import Html.Keyed as Keyed
import Html.Lazy exposing (lazy, lazy2)
import Http
import Json.Encode as Encode
import Json.Decode as Decode
import Json.Decode.Pipeline as Pipeline
import Task
import WebSocket


main : Program (Maybe Model) Model Msg
main =
    Html.programWithFlags
        { init = init
        , view = view
        , update = updateWithStorage
        , subscriptions = subscriptions
        }


port setStorage : Model -> Cmd msg


{-| Middleware for update to save the state in local storage on every update
using the setStorage port.
-}
updateWithStorage : Msg -> Model -> ( Model, Cmd Msg )
updateWithStorage msg model =
    let
        ( newModel, cmds ) =
            update msg model
    in
        ( newModel
        , Cmd.batch [ setStorage newModel, cmds ]
        )



-- MODEL


type alias Model =
    { list : TodoList
    , backendList : TodoList
    , input : String
    , nextID : Int
    , visibility : String
    , error : Maybe String
    }


emptyModel : Model
emptyModel =
    { list = newList []
    , backendList = newList []
    , input = ""
    , nextID = 0
    , visibility = "All"
    , error = Nothing
    }


init : Maybe Model -> ( Model, Cmd Msg )
init savedModel =
    Maybe.withDefault emptyModel savedModel
        ! [ getTodoLists ]



-- UPDATE


type Msg
    = NoOp
      -- Emitted by the event bus, via the web socket. Fetches all items.
    | Event String
    | StoreInput String
    | ChangeVisibility String
    | GetTodoListsResp (Result Http.Error (List TodoList))
    | AddItem
    | AddItemResp (Result Http.Error ())
    | RemoveItem Int
    | RemoveItemResp (Result Http.Error ())
    | RemoveCompleted
    | RemoveCompletedResp (Result Http.Error ())
    | SetEditing Int Bool
    | SetItemDescription Int String
    | SetItemDescriptionResp (Result Http.Error ())
    | CheckItem Int Bool
    | CheckItemResp (Result Http.Error ())
    | CheckAllItems Bool
    | CheckAllItemsResp (Result Http.Error ())


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        NoOp ->
            model ! []

        Event event ->
            model
                ! [ getTodoLists ]

        GetTodoListsResp (Ok lists) ->
            let
                list =
                    case lists of
                        [] ->
                            -- TODO: Create a list if none exists.
                            newList []

                        x :: _ ->
                            -- NOTE: Always use the first todo list for now.
                            x
            in
                { model | list = list }
                    ! []

        GetTodoListsResp (Err _) ->
            model ! []

        StoreInput str ->
            { model | input = str }
                ! []

        ChangeVisibility visibility ->
            { model | visibility = visibility }
                ! []

        AddItem ->
            { model | error = Nothing }
                ! [ postAddItem model.list.id model.input ]

        AddItemResp (Ok _) ->
            { model | input = "" }
                ! []

        AddItemResp (Err error) ->
            { model | error = Just (toString error) }
                ! []

        RemoveItem itemID ->
            { model | error = Nothing }
                ! [ postRemoveItem model.list.id itemID ]

        RemoveItemResp (Ok ()) ->
            model ! []

        RemoveItemResp (Err error) ->
            { model | error = Just (toString error) }
                ! []

        RemoveCompleted ->
            { model | error = Nothing }
                ! [ postRemoveCompleted model.list.id ]

        RemoveCompletedResp (Ok ()) ->
            model ! []

        RemoveCompletedResp (Err error) ->
            { model | error = Just (toString error) }
                ! []

        SetEditing id isEditing ->
            -- TODO: When isEditing goes false, send a post to set desc.
            -- This is to avoid sending commands on every keystroke.
            let
                focusTask =
                    Dom.focus ("todo-" ++ toString id)
            in
                { model | list = setEditing model.list id isEditing }
                    ! [ Task.attempt (\_ -> NoOp) focusTask ]

        SetItemDescription itemID desc ->
            { model | error = Nothing }
                ! [ postSetItemDescription model.list.id itemID desc ]

        SetItemDescriptionResp (Ok ()) ->
            model ! []

        SetItemDescriptionResp (Err error) ->
            { model | error = Just (toString error) }
                ! []

        CheckItem itemID isChecked ->
            { model | error = Nothing }
                ! [ postCheckItem model.list.id itemID isChecked ]

        CheckItemResp (Ok ()) ->
            model ! []

        CheckItemResp (Err error) ->
            { model | error = Just (toString error) }
                ! []

        CheckAllItems isChecked ->
            { model | error = Nothing }
                ! [ postCheckAllItems model.list.id isChecked ]

        CheckAllItemsResp (Ok ()) ->
            model ! []

        CheckAllItemsResp (Err error) ->
            { model | error = Just (toString error) }
                ! []



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    -- Listen to faked backend events.
    WebSocket.listen "ws://localhost:8080/api/events/" Event



-- TYPES


type alias TodoList =
    { id : String
    , version : Int
    , items : List TodoItem
    , createdAt : String
    , updatedAt : String
    }


type alias TodoItem =
    { id : Int
    , description : String
    , completed : Bool
    , editing : Bool
    }


newList : List TodoItem -> TodoList
newList items =
    { id = ""
    , version = 0
    , items = items
    , createdAt = ""
    , updatedAt = ""
    }


setEditing : TodoList -> Int -> Bool -> TodoList
setEditing list id isEditing =
    let
        updateItem item =
            if item.id == id then
                { item | editing = isEditing }
            else
                item
    in
        { list | items = List.map updateItem list.items }



-- API


getTodoLists : Cmd Msg
getTodoLists =
    Http.send GetTodoListsResp
        (Http.get
            "http://localhost:8080/api/todos/"
            (Decode.list decodeTodoList)
        )


postAddItem : String -> String -> Cmd Msg
postAddItem id desc =
    if String.isEmpty desc then
        Cmd.none
    else
        postCmd "add_item"
            [ ( "id", Encode.string id )
            , ( "desc", Encode.string desc )
            ]


postRemoveItem : String -> Int -> Cmd Msg
postRemoveItem id itemID =
    postCmd "remove_item"
        [ ( "id", Encode.string id )
        , ( "item_id", Encode.int itemID )
        ]


postRemoveCompleted : String -> Cmd Msg
postRemoveCompleted id =
    postCmd "remove_completed"
        [ ( "id", Encode.string id )
        ]


postSetItemDescription : String -> Int -> String -> Cmd Msg
postSetItemDescription id itemID desc =
    postCmd "set_item_desc"
        [ ( "id", Encode.string id )
        , ( "item_id", Encode.int itemID )
        , ( "desc", Encode.string desc )
        ]


postCheckItem : String -> Int -> Bool -> Cmd Msg
postCheckItem id itemID isChecked =
    postCmd "check_item"
        [ ( "id", Encode.string id )
        , ( "item_id", Encode.int itemID )
        , ( "checked", Encode.bool isChecked )
        ]


postCheckAllItems : String -> Bool -> Cmd Msg
postCheckAllItems id isChecked =
    postCmd "check_all_items"
        [ ( "id", Encode.string id )
        , ( "checked", Encode.bool isChecked )
        ]


postCmd : String -> List ( String, Encode.Value ) -> Cmd Msg
postCmd cmd body =
    Http.send AddItemResp
        (Http.request
            { method = "POST"
            , headers = []
            , url = "http://localhost:8080/api/todos/" ++ cmd
            , body = Http.jsonBody (Encode.object body)
            , expect = Http.expectStringResponse (\_ -> Ok ())
            , timeout = Nothing
            , withCredentials = False
            }
        )



-- JSON Decoding


decodeTodoList : Decode.Decoder TodoList
decodeTodoList =
    Pipeline.decode TodoList
        |> Pipeline.required "id" Decode.string
        |> Pipeline.required "version" Decode.int
        |> Pipeline.required "items" (Decode.list decodeTodoItem)
        |> Pipeline.required "created_at" Decode.string
        |> Pipeline.required "updated_at" Decode.string


decodeTodoItem : Decode.Decoder TodoItem
decodeTodoItem =
    Pipeline.decode TodoItem
        |> Pipeline.required "id" Decode.int
        |> Pipeline.required "desc" Decode.string
        |> Pipeline.required "completed" Decode.bool
        |> Pipeline.hardcoded False



-- VIEW


view : Model -> Html Msg
view model =
    div
        [ class "todomvc-wrapper"
        , style [ ( "visibility", "hidden" ) ]
        ]
        [ section
            [ class "todoapp" ]
            [ lazy viewInput model.input
            , lazy2 viewItems model.visibility model.list.items
            , lazy2 viewControls model.visibility model.list.items
            ]
        , viewFooter
        , text (toString model.error)
        ]


viewInput : String -> Html Msg
viewInput task =
    header
        [ class "header" ]
        [ h1 [] [ text "todos" ]
        , input
            [ class "new-todo"
            , placeholder "What needs to be done?"
            , autofocus True
            , value task
            , name "newTodo"
            , onInput StoreInput
            , onEnter AddItem
            ]
            []
        ]


onEnter : Msg -> Attribute Msg
onEnter msg =
    let
        isEnter code =
            if code == 13 then
                Decode.succeed msg
            else
                Decode.fail "not ENTER"
    in
        on "keydown" (Decode.andThen isEnter keyCode)



-- VIEW ALL ENTRIES


viewItems : String -> List TodoItem -> Html Msg
viewItems visibility items =
    let
        isVisible todo =
            case visibility of
                "Completed" ->
                    todo.completed

                "Active" ->
                    not todo.completed

                _ ->
                    True

        allCompleted =
            List.all .completed items

        cssVisibility =
            if List.isEmpty items then
                "hidden"
            else
                "visible"
    in
        section
            [ class "main"
            , style [ ( "visibility", cssVisibility ) ]
            ]
            [ input
                [ class "toggle-all"
                , type_ "checkbox"
                , name "toggle"
                , checked allCompleted
                , onClick (CheckAllItems (not allCompleted))
                ]
                []
            , label
                [ for "toggle-all" ]
                [ text "Mark all as complete" ]
            , Keyed.ul [ class "todo-list" ] <|
                List.map viewKeyedItem (List.filter isVisible items)
            ]



-- VIEW INDIVIDUAL ENTRIES


viewKeyedItem : TodoItem -> ( String, Html Msg )
viewKeyedItem todo =
    ( toString todo.id, lazy viewItem todo )


viewItem : TodoItem -> Html Msg
viewItem todo =
    li
        [ classList [ ( "completed", todo.completed ), ( "editing", todo.editing ) ] ]
        [ div
            [ class "view" ]
            [ input
                [ class "toggle"
                , type_ "checkbox"
                , checked todo.completed
                , onClick (CheckItem todo.id (not todo.completed))
                ]
                []
            , label
                [ onDoubleClick (SetEditing todo.id True) ]
                [ text todo.description ]
            , button
                [ class "destroy"
                , onClick (RemoveItem todo.id)
                ]
                []
            ]
        , input
            [ class "edit"
            , value todo.description
            , name "title"
            , id ("todo-" ++ toString todo.id)
            , onInput (SetItemDescription todo.id)
            , onBlur (SetEditing todo.id False)
            , onEnter (SetEditing todo.id False)
            ]
            []
        ]



-- VIEW CONTROLS AND FOOTER


viewControls : String -> List TodoItem -> Html Msg
viewControls visibility items =
    let
        itemsCompleted =
            List.length (List.filter .completed items)

        itemsLeft =
            List.length items - itemsCompleted
    in
        footer
            [ class "footer"
            , hidden (List.isEmpty items)
            ]
            [ lazy viewControlsCount itemsLeft
            , lazy viewControlsFilters visibility
            , lazy viewControlsClear itemsCompleted
            ]


viewControlsCount : Int -> Html Msg
viewControlsCount itemsLeft =
    let
        item_ =
            if itemsLeft == 1 then
                " item"
            else
                " items"
    in
        span
            [ class "todo-count" ]
            [ strong [] [ text (toString itemsLeft) ]
            , text (item_ ++ " left")
            ]


viewControlsFilters : String -> Html Msg
viewControlsFilters visibility =
    ul
        [ class "filters" ]
        [ visibilitySwap "#/" "All" visibility
        , text " "
        , visibilitySwap "#/active" "Active" visibility
        , text " "
        , visibilitySwap "#/completed" "Completed" visibility
        ]


visibilitySwap : String -> String -> String -> Html Msg
visibilitySwap uri visibility actualVisibility =
    li
        [ onClick (ChangeVisibility visibility) ]
        [ a [ href uri, classList [ ( "selected", visibility == actualVisibility ) ] ]
            [ text visibility ]
        ]


viewControlsClear : Int -> Html Msg
viewControlsClear itemsCompleted =
    button
        [ class "clear-completed"
        , hidden (itemsCompleted == 0)
        , onClick RemoveCompleted
        ]
        [ text ("Clear completed (" ++ toString itemsCompleted ++ ")")
        ]


viewFooter : Html msg
viewFooter =
    footer [ class "info" ]
        [ p [] [ text "Double-click to edit a todo" ]
        , p []
            [ text "Written by "
            , a [ href "https://github.com/evancz" ] [ text "Evan Czaplicki" ]
            ]
        , p []
            [ text "Part of "
            , a [ href "http://todomvc.com" ] [ text "TodoMVC" ]
            ]
        ]

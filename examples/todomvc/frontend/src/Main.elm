port module Main exposing (..)

{-| TodoMVC implemented in Elm, using plain HTML and CSS for rendering.

This application is broken up into three key parts:

1.  Model - a full definition of the application's state
2.  Update - a way to step the application state forward
3.  View - a way to visualize our application state with HTML

This clean division of concerns is a core part of Elm. You can read more about
this in <http://guide.elm-lang.org/architecture/index.html>

-}

import Browser
import Browser.Dom as Dom
import Debug
import Html exposing (..)
import Html.Attributes exposing (..)
import Html.Events exposing (..)
import Html.Keyed as Keyed
import Html.Lazy exposing (lazy, lazy2)
import Http
import Json.Decode as Decode
import Json.Decode.Pipeline as Pipeline
import Json.Encode as Encode
import Task
import VirtualDom



-- MAIN


main : Program (Maybe Model) Model Msg
main =
    Browser.document
        { init = init
        , view = \model -> { title = "Event Horizon • TodoMVC", body = [ view model ] }
        , update = update
        , subscriptions = subscriptions
        }



-- PORTS


port eventReceiver : (String -> msg) -> Sub msg



-- MODEL
-- The full application state of our todo app.


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
init maybeModel =
    ( emptyModel
    , getTodoLists
    )



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
            ( model, Cmd.none )

        Event event ->
            ( model
            , getTodoLists
            )

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
            ( { model | list = list }
            , Cmd.none
            )

        GetTodoListsResp (Err _) ->
            ( model, Cmd.none )

        StoreInput str ->
            ( { model | input = str }
            , Cmd.none
            )

        ChangeVisibility visibility ->
            ( { model | visibility = visibility }
            , Cmd.none
            )

        AddItem ->
            ( { model | error = Nothing }
            , postAddItem model.list.id model.input
            )

        AddItemResp (Ok _) ->
            ( { model | input = "" }
            , Cmd.none
            )

        AddItemResp (Err error) ->
            ( { model | error = Just (errorToString error) }
            , Cmd.none
            )

        RemoveItem itemID ->
            ( { model | error = Nothing }
            , postRemoveItem model.list.id itemID
            )

        RemoveItemResp (Ok ()) ->
            ( model, Cmd.none )

        RemoveItemResp (Err error) ->
            ( { model | error = Just (errorToString error) }
            , Cmd.none
            )

        RemoveCompleted ->
            ( { model | error = Nothing }
            , postRemoveCompleted model.list.id
            )

        RemoveCompletedResp (Ok ()) ->
            ( model, Cmd.none )

        RemoveCompletedResp (Err error) ->
            ( { model | error = Just (errorToString error) }
            , Cmd.none
            )

        SetEditing id isEditing ->
            -- TODO: When isEditing goes false, send a post to set desc.
            -- This is to avoid sending commands on every keystroke.
            let
                focus =
                    Dom.focus ("todo-" ++ String.fromInt id)
            in
            ( { model | list = setEditing model.list id isEditing }
            , Task.attempt (\_ -> NoOp) focus
            )

        SetItemDescription itemID desc ->
            ( { model | error = Nothing }
            , postSetItemDescription model.list.id itemID desc
            )

        SetItemDescriptionResp (Ok ()) ->
            ( model, Cmd.none )

        SetItemDescriptionResp (Err error) ->
            ( { model | error = Just (errorToString error) }
            , Cmd.none
            )

        CheckItem itemID isChecked ->
            ( { model | error = Nothing }
            , postCheckItem model.list.id itemID isChecked
            )

        CheckItemResp (Ok ()) ->
            ( model, Cmd.none )

        CheckItemResp (Err error) ->
            ( { model | error = Just (errorToString error) }
            , Cmd.none
            )

        CheckAllItems isChecked ->
            ( { model | error = Nothing }
            , postCheckAllItems model.list.id isChecked
            )

        CheckAllItemsResp (Ok ()) ->
            ( model, Cmd.none )

        CheckAllItemsResp (Err error) ->
            ( { model | error = Just (errorToString error) }
            , Cmd.none
            )



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    let
        a =
            Debug.log (Debug.toString Event)
    in
    -- Listen to backend events.
    eventReceiver Event



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
    Http.get
        { url = "http://localhost:8080/api/todos/"
        , expect = Http.expectJson GetTodoListsResp (Decode.list decodeTodoList)
        }


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
    Http.post
        { url = "http://localhost:8080/api/todos/" ++ cmd
        , body = Http.jsonBody (Encode.object body)
        , expect = Http.expectWhatever AddItemResp
        }



-- JSON decoding


decodeTodoList : Decode.Decoder TodoList
decodeTodoList =
    Decode.succeed TodoList
        |> Pipeline.required "id" Decode.string
        |> Pipeline.required "version" Decode.int
        |> Pipeline.required "items" (Decode.list decodeTodoItem)
        |> Pipeline.required "created_at" Decode.string
        |> Pipeline.required "updated_at" Decode.string


decodeTodoItem : Decode.Decoder TodoItem
decodeTodoItem =
    Decode.succeed TodoItem
        |> Pipeline.required "id" Decode.int
        |> Pipeline.required "desc" Decode.string
        |> Pipeline.required "completed" Decode.bool
        |> Pipeline.hardcoded False



-- Error formatting


errorToString : Http.Error -> String
errorToString error =
    case error of
        Http.BadUrl url ->
            "The URL " ++ url ++ " was invalid"

        Http.Timeout ->
            "Unable to reach the server, try again"

        Http.NetworkError ->
            "Unable to reach the server, check your network connection"

        Http.BadStatus 500 ->
            "The server had a problem, try again later"

        Http.BadStatus 400 ->
            "Verify your information and try again"

        Http.BadStatus _ ->
            "Unknown error"

        Http.BadBody errorMessage ->
            errorMessage



-- VIEW


view : Model -> Html Msg
view model =
    div
        []
        [ section
            [ class "todoapp" ]
            [ lazy viewInput model.input
            , lazy2 viewItems model.visibility model.list.items
            , lazy2 viewControls model.visibility model.list.items
            ]
        , viewFooter
        , text (Maybe.withDefault "" model.error)
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
        , style "visibility" cssVisibility
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
    ( String.fromInt todo.id, lazy viewItem todo )


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
            , id ("todo-" ++ String.fromInt todo.id)
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
        [ strong [] [ text (String.fromInt itemsLeft) ]
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
        [ text ("Clear completed (" ++ String.fromInt itemsCompleted ++ ")")
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
